#!/usr/bin/env bash

################################################################################
# generate-subtitles.sh
#
# Converts course script timestamps to SRT subtitle format
#
# Usage:
#   ./generate-subtitles.sh <script_file.md> [output.srt]
#
# Example:
#   ./generate-subtitles.sh ../Course_01_Introduction/01_Welcome.md ../videos/course_01/chapter_01.srt
#
# SRT Format:
#   1
#   00:00:00,000 --> 00:00:30,000
#   Subtitle text here
#
# Dependencies:
#   - bash 4.0+
#   - standard Unix tools (awk, sed)
################################################################################

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Convert MM:SS timestamp to SRT format (HH:MM:SS,mmm)
convert_timestamp() {
    local timestamp=$1

    # Handle both MM:SS and HH:MM:SS formats
    if [[ $timestamp =~ ^([0-9]{2}):([0-9]{2})$ ]]; then
        # MM:SS format
        echo "00:${BASH_REMATCH[1]}:${BASH_REMATCH[2]},000"
    elif [[ $timestamp =~ ^([0-9]{2}):([0-9]{2}):([0-9]{2})$ ]]; then
        # HH:MM:SS format
        echo "${BASH_REMATCH[1]}:${BASH_REMATCH[2]}:${BASH_REMATCH[3]},000"
    else
        error "Invalid timestamp format: $timestamp (expected MM:SS or HH:MM:SS)"
    fi
}

# Extract script content and timestamps
extract_script_content() {
    local script_file=$1
    local temp_file=$(mktemp)

    info "Extracting script content from $script_file..."

    # Extract content between "## Video Script" and "## Slide Outline"
    awk '/## Video Script/,/## Slide Outline/ {
        if (!/## Video Script/ && !/## Slide Outline/) print
    }' "$script_file" > "$temp_file"

    if [ ! -s "$temp_file" ]; then
        error "No video script found in file"
    fi

    echo "$temp_file"
}

# Parse script and generate SRT
generate_srt() {
    local script_content=$1
    local output_srt=$2

    info "Generating SRT subtitles..."

    local subtitle_num=1
    local current_start=""
    local current_end=""
    local current_text=""
    local in_segment=false

    > "$output_srt"  # Clear output file

    while IFS= read -r line; do
        # Check for timestamp pattern [MM:SS - MM:SS]
        if [[ $line =~ \[([0-9]{2}:[0-9]{2})\ -\ ([0-9]{2}:[0-9]{2})\] ]]; then
            # If we have accumulated text, write it out
            if [ "$in_segment" = true ] && [ -n "$current_text" ]; then
                echo "$subtitle_num" >> "$output_srt"
                echo "$(convert_timestamp "$current_start") --> $(convert_timestamp "$current_end")" >> "$output_srt"
                echo "$current_text" >> "$output_srt"
                echo "" >> "$output_srt"
                ((subtitle_num++))
            fi

            # Start new segment
            current_start="${BASH_REMATCH[1]}"
            current_end="${BASH_REMATCH[2]}"
            current_text=""
            in_segment=true

            # Extract section title if present
            local section_title=$(echo "$line" | sed -E 's/\[[0-9:\ \-]+\]\ *//')
            if [ -n "$section_title" ]; then
                current_text="$section_title"
            fi

        elif [ "$in_segment" = true ] && [ -n "$line" ]; then
            # Accumulate text for current segment
            # Skip lines that are section headers (all caps)
            if [[ ! $line =~ ^\*\* ]] && [[ ! $line =~ ^# ]]; then
                if [ -n "$current_text" ]; then
                    current_text="$current_text $line"
                else
                    current_text="$line"
                fi
            fi
        fi
    done < "$script_content"

    # Write last segment
    if [ "$in_segment" = true ] && [ -n "$current_text" ]; then
        echo "$subtitle_num" >> "$output_srt"
        echo "$(convert_timestamp "$current_start") --> $(convert_timestamp "$current_end")" >> "$output_srt"
        echo "$current_text" >> "$output_srt"
        echo "" >> "$output_srt"
    fi

    info "✓ Generated $(($subtitle_num - 1)) subtitle segments"
}

# Split long subtitles into multiple lines
split_long_lines() {
    local srt_file=$1
    local max_chars=80
    local max_lines=2

    info "Formatting subtitle line lengths..."

    # Create temporary file
    local temp_file=$(mktemp)

    awk -v max_chars="$max_chars" -v max_lines="$max_lines" '
        /^[0-9]+$/ || /-->/ { print; next }
        /^$/ { print; next }
        {
            text = $0
            if (length(text) <= max_chars) {
                print text
            } else {
                # Split into multiple lines at word boundaries
                words_count = split(text, words, " ")
                line = ""
                for (i = 1; i <= words_count; i++) {
                    if (length(line " " words[i]) <= max_chars) {
                        if (line == "") line = words[i]
                        else line = line " " words[i]
                    } else {
                        print line
                        line = words[i]
                    }
                }
                if (line != "") print line
            }
        }
    ' "$srt_file" > "$temp_file"

    mv "$temp_file" "$srt_file"
}

# Validate SRT format
validate_srt() {
    local srt_file=$1

    info "Validating SRT format..."

    local errors=0

    # Check for required patterns
    if ! grep -q "^[0-9]\+$" "$srt_file"; then
        warn "No subtitle numbers found"
        ((errors++))
    fi

    if ! grep -q "-->" "$srt_file"; then
        warn "No timestamp ranges found"
        ((errors++))
    fi

    if [ $errors -eq 0 ]; then
        info "✓ SRT validation passed"
    else
        warn "SRT validation found $errors potential issues"
    fi
}

# Generate VTT format (WebVTT for web players)
generate_vtt() {
    local srt_file=$1
    local vtt_file="${srt_file%.srt}.vtt"

    info "Generating WebVTT format..."

    echo "WEBVTT" > "$vtt_file"
    echo "" >> "$vtt_file"

    # Convert SRT to VTT (replace comma with period in timestamps)
    sed 's/\([0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}\),\([0-9]\{3\}\)/\1.\2/g' "$srt_file" >> "$vtt_file"

    info "✓ WebVTT generated: $vtt_file"
}

# Main function
main() {
    if [ $# -lt 1 ]; then
        error "Usage: $0 <script_file.md> [output.srt]"
    fi

    local script_file=$1
    local output_srt=${2:-"${script_file%.md}.srt"}

    if [ ! -f "$script_file" ]; then
        error "Script file not found: $script_file"
    fi

    info "Processing: $script_file"
    info "Output: $output_srt"

    # Create output directory if needed
    mkdir -p "$(dirname "$output_srt")"

    # Extract script content
    local script_temp=$(extract_script_content "$script_file")

    # Generate SRT
    generate_srt "$script_temp" "$output_srt"

    # Format and validate
    split_long_lines "$output_srt"
    validate_srt "$output_srt"

    # Generate VTT
    generate_vtt "$output_srt"

    # Cleanup
    rm -f "$script_temp"

    info "✓ Subtitle generation complete!"
    info "SRT file: $output_srt"
    info "VTT file: ${output_srt%.srt}.vtt"

    # Show statistics
    local subtitle_count=$(grep -c "^[0-9]\+$" "$output_srt" || echo "0")
    local total_text=$(grep -v "^[0-9]\+$" "$output_srt" | grep -v "^$" | grep -v "-->" | wc -l)

    info "Statistics:"
    info "  - Subtitle segments: $subtitle_count"
    info "  - Text lines: $total_text"
}

# Run main function
main "$@"
