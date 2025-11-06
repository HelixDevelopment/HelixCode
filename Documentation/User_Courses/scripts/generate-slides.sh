#!/usr/bin/env bash

################################################################################
# generate-slides.sh
#
# Converts Markdown course scripts to reveal.js slide presentations
#
# Usage:
#   ./generate-slides.sh <script_file.md> [output_dir]
#
# Example:
#   ./generate-slides.sh ../Course_01_Introduction/01_Welcome.md ../Course_01_Introduction/slides/
#
# Dependencies:
#   - pandoc (for Markdown to HTML conversion)
#   - reveal.js (for presentation framework)
#
# Output:
#   - HTML presentation file compatible with reveal.js
#   - Embedded slide content from "Slide Outline" sections
################################################################################

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
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

# Check dependencies
check_dependencies() {
    info "Checking dependencies..."

    if ! command -v pandoc &> /dev/null; then
        error "pandoc is not installed. Install with: brew install pandoc (macOS) or apt-get install pandoc (Linux)"
    fi

    if [ ! -d "reveal.js" ]; then
        warn "reveal.js not found in current directory. Downloading..."
        git clone https://github.com/hakimel/reveal.js.git
    fi

    info "All dependencies satisfied"
}

# Extract slide outline from script
extract_slides() {
    local script_file=$1
    local temp_file=$(mktemp)

    info "Extracting slide outline from $script_file..."

    # Extract content between "## Slide Outline" and next "##" heading
    awk '/## Slide Outline/,/^## [^S]/ {
        if (!/## Slide Outline/ && !/^## [^S]/) print
    }' "$script_file" > "$temp_file"

    if [ ! -s "$temp_file" ]; then
        warn "No slide outline found in script. Will generate basic slides from headings."
        # Generate basic slides from main content
        awk '/## Video Script/,/## Slide Outline/ {
            if (!/## Video Script/ && !/## Slide Outline/) print
        }' "$script_file" > "$temp_file"
    fi

    echo "$temp_file"
}

# Convert to reveal.js format
convert_to_revealjs() {
    local slides_md=$1
    local output_html=$2
    local title=$3

    info "Converting to reveal.js presentation..."

    # Create reveal.js presentation
    pandoc "$slides_md" \
        -t revealjs \
        -s \
        -o "$output_html" \
        --slide-level=2 \
        -V theme=black \
        -V transition=slide \
        -V width=1920 \
        -V height=1080 \
        --metadata title="$title"

    info "Presentation generated: $output_html"
}

# Generate standalone HTML with embedded reveal.js
generate_standalone() {
    local slides_md=$1
    local output_html=$2
    local title=$3

    info "Generating standalone HTML presentation..."

    cat > "$output_html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TITLE_PLACEHOLDER</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@4.5.0/dist/reveal.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@4.5.0/dist/theme/black.css">
    <style>
        .reveal h1, .reveal h2, .reveal h3 { text-transform: none; }
        .reveal pre { font-size: 0.6em; }
        .reveal code { background: #3f3f3f; padding: 2px 8px; border-radius: 3px; }
        .reveal ul { display: block; }
        .reveal ol { display: block; }
    </style>
</head>
<body>
    <div class="reveal">
        <div class="slides">
EOF

    # Insert content placeholder
    echo "CONTENT_PLACEHOLDER" >> "$output_html"

    cat >> "$output_html" << 'EOF'
        </div>
    </div>
    <script src="https://cdn.jsdelivr.net/npm/reveal.js@4.5.0/dist/reveal.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/reveal.js@4.5.0/plugin/notes/notes.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/reveal.js@4.5.0/plugin/highlight/highlight.js"></script>
    <script>
        Reveal.initialize({
            hash: true,
            width: 1920,
            height: 1080,
            transition: 'slide',
            plugins: [ RevealNotes, RevealHighlight ]
        });
    </script>
</body>
</html>
EOF

    # Convert markdown slides to HTML sections
    local temp_html=$(mktemp)

    # Convert each slide to a section
    awk '
        /^\*\*Slide [0-9]+:/ {
            if (in_slide) print "</section>"
            print "<section>"
            in_slide = 1
            next
        }
        /^$/ && in_slide { next }
        in_slide { print }
        END { if (in_slide) print "</section>" }
    ' "$slides_md" | while IFS= read -r line; do
        # Convert markdown to HTML
        if [[ $line == "**"*"**" ]]; then
            # Header
            echo "$line" | sed 's/\*\*\(.*\)\*\*/<h2>\1<\/h2>/'
        elif [[ $line == "- "* ]]; then
            # List item
            echo "$line" | sed 's/^- /<li>/' | sed 's/$/<\/li>/'
        else
            echo "<p>$line</p>"
        fi
    done > "$temp_html"

    # Replace placeholders
    sed -i.bak "s/TITLE_PLACEHOLDER/$title/" "$output_html"
    sed -i.bak "/CONTENT_PLACEHOLDER/r $temp_html" "$output_html"
    sed -i.bak "/CONTENT_PLACEHOLDER/d" "$output_html"

    rm -f "$output_html.bak" "$temp_html"

    info "Standalone presentation generated: $output_html"
}

# Main function
main() {
    if [ $# -lt 1 ]; then
        error "Usage: $0 <script_file.md> [output_dir]"
    fi

    local script_file=$1
    local output_dir=${2:-.}

    if [ ! -f "$script_file" ]; then
        error "Script file not found: $script_file"
    fi

    # Extract chapter title
    local title=$(grep "^# " "$script_file" | head -1 | sed 's/^# //')
    if [ -z "$title" ]; then
        title="HelixCode Course Slides"
    fi

    info "Generating slides for: $title"

    # Check dependencies
    check_dependencies

    # Create output directory
    mkdir -p "$output_dir"

    # Extract slides
    local slides_temp=$(extract_slides "$script_file")

    # Generate output filename
    local base_name=$(basename "$script_file" .md)
    local output_html="$output_dir/${base_name}_slides.html"

    # Generate standalone HTML
    generate_standalone "$slides_temp" "$output_html" "$title"

    # Cleanup
    rm -f "$slides_temp"

    info "✓ Slide generation complete!"
    info "Open in browser: file://$(realpath "$output_html")"
}

# Run main function
main "$@"
