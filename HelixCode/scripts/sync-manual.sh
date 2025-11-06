#!/bin/bash
#
# sync-manual.sh - Sync user manual to GitHub Pages Website
#
# This script synchronizes the user manual from Documentation/User_Manual to
# the GitHub Pages Website, generates PDF versions, and logs all operations.
#
# Usage: ./scripts/sync-manual.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MANUAL_SOURCE_DIR="$PROJECT_ROOT/Documentation/User_Manual"
GITHUB_PAGES_DIR="$(dirname "$PROJECT_ROOT")/Github-Pages-Website"
MANUAL_DEST_DIR="$GITHUB_PAGES_DIR/docs/manual"
LOG_FILE="$SCRIPT_DIR/sync-manual.log"

# Timestamp
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
TIMESTAMP_COMPACT=$(date '+%Y%m%d_%H%M%S')

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "[$TIMESTAMP] [INFO] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    echo "[$TIMESTAMP] [SUCCESS] $1" >> "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    echo "[$TIMESTAMP] [WARN] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    echo "[$TIMESTAMP] [ERROR] $1" >> "$LOG_FILE"
}

# Main sync process
main() {
    log_info "=== HelixCode Manual Sync Starting ==="
    log_info "Source: $MANUAL_SOURCE_DIR"
    log_info "Destination: $MANUAL_DEST_DIR"

    # Check if source directory exists
    if [ ! -d "$MANUAL_SOURCE_DIR" ]; then
        log_error "Source directory does not exist: $MANUAL_SOURCE_DIR"
        exit 1
    fi

    # Check if GitHub Pages directory exists
    if [ ! -d "$GITHUB_PAGES_DIR" ]; then
        log_error "GitHub Pages directory does not exist: $GITHUB_PAGES_DIR"
        log_warn "Please ensure the GitHub Pages Website repository is cloned"
        exit 1
    fi

    # Create destination directories
    log_info "Creating destination directories..."
    mkdir -p "$MANUAL_DEST_DIR"
    mkdir -p "$MANUAL_DEST_DIR/images"

    # Copy manual.html to index.html
    log_info "Copying manual.html to GitHub Pages..."
    if [ -f "$MANUAL_SOURCE_DIR/manual.html" ]; then
        cp "$MANUAL_SOURCE_DIR/manual.html" "$MANUAL_DEST_DIR/index.html"
        log_success "Copied manual.html to index.html"
    else
        log_error "manual.html not found in $MANUAL_SOURCE_DIR"
        log_warn "Run md-to-html first to generate the manual"
        exit 1
    fi

    # Update timestamp in the HTML file
    log_info "Adding generation timestamp..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|Generated on: <span id=\"timestamp\"></span>|Generated on: $TIMESTAMP|g" "$MANUAL_DEST_DIR/index.html"
    else
        # Linux
        sed -i "s|Generated on: <span id=\"timestamp\"></span>|Generated on: $TIMESTAMP|g" "$MANUAL_DEST_DIR/index.html"
    fi
    log_success "Timestamp updated"

    # Copy images if they exist
    log_info "Copying images..."
    if [ -d "$MANUAL_SOURCE_DIR/images" ] && [ "$(ls -A $MANUAL_SOURCE_DIR/images 2>/dev/null)" ]; then
        cp -r "$MANUAL_SOURCE_DIR/images/"* "$MANUAL_DEST_DIR/images/" 2>/dev/null || true
        IMAGE_COUNT=$(ls -1 "$MANUAL_SOURCE_DIR/images" 2>/dev/null | wc -l | tr -d ' ')
        log_success "Copied $IMAGE_COUNT image(s)"
    else
        log_warn "No images found in $MANUAL_SOURCE_DIR/images/"
    fi

    # Generate PDF version using wkhtmltopdf (if available)
    log_info "Checking for PDF generation tools..."
    if command -v wkhtmltopdf &> /dev/null; then
        log_info "Generating PDF version..."
        PDF_OUTPUT="$MANUAL_DEST_DIR/HelixCode_User_Manual_${TIMESTAMP_COMPACT}.pdf"

        wkhtmltopdf \
            --enable-local-file-access \
            --margin-top 20mm \
            --margin-bottom 20mm \
            --margin-left 15mm \
            --margin-right 15mm \
            --print-media-type \
            "$MANUAL_DEST_DIR/index.html" \
            "$PDF_OUTPUT" 2>&1 | tee -a "$LOG_FILE" || log_warn "PDF generation had warnings"

        if [ -f "$PDF_OUTPUT" ]; then
            # Create symlink to latest PDF
            ln -sf "$(basename $PDF_OUTPUT)" "$MANUAL_DEST_DIR/HelixCode_User_Manual_Latest.pdf"
            PDF_SIZE=$(du -h "$PDF_OUTPUT" | cut -f1)
            log_success "PDF generated: $PDF_OUTPUT ($PDF_SIZE)"
        else
            log_warn "PDF generation failed"
        fi
    else
        log_warn "wkhtmltopdf not installed - skipping PDF generation"
        log_warn "Install with: brew install --cask wkhtmltopdf (macOS)"
    fi

    # Generate README for manual directory
    log_info "Generating README..."
    cat > "$MANUAL_DEST_DIR/README.md" << 'EOFREADME'
# HelixCode User Manual

This directory contains the HelixCode user manual in HTML and PDF formats.

## Files

- **index.html**: Interactive HTML version with search, navigation, and dark mode
- **HelixCode_User_Manual_Latest.pdf**: Latest PDF version (if generated)
- **images/**: Supporting images and assets

## Features

- Full-text search across documentation
- Sidebar navigation with table of contents
- Responsive mobile-friendly design
- Light/Dark theme toggle
- Syntax highlighting for code examples
- Interactive anchor links

## Viewing

### Online
Visit the manual at: https://your-username.github.io/Github-Pages-Website/docs/manual/

### Local
Open `index.html` in any modern web browser.

### PDF
Download the PDF version for offline reading.

---

*Automatically generated and synced from the HelixCode repository.*
EOFREADME

    log_success "README.md generated"

    # Generate sync metadata
    cat > "$MANUAL_DEST_DIR/.sync-metadata.json" << EOFMETA
{
  "last_sync": "$TIMESTAMP",
  "source": "$MANUAL_SOURCE_DIR",
  "destination": "$MANUAL_DEST_DIR",
  "version": "1.0",
  "sync_script": "$0"
}
EOFMETA

    log_success "Sync metadata generated"

    # Display summary
    echo ""
    log_success "=== Sync Complete ==="
    HTML_SIZE=$(du -h "$MANUAL_DEST_DIR/index.html" | cut -f1)
    log_info "HTML size: $HTML_SIZE"

    if [ -f "$MANUAL_DEST_DIR/HelixCode_User_Manual_Latest.pdf" ]; then
        PDF_SIZE=$(du -h "$MANUAL_DEST_DIR/HelixCode_User_Manual_Latest.pdf" | cut -f1)
        log_info "PDF size: $PDF_SIZE"
    fi

    # Check for git changes
    echo ""
    log_info "Checking for changes in GitHub Pages repository..."
    cd "$GITHUB_PAGES_DIR"

    if git diff --quiet && git diff --cached --quiet; then
        log_info "No changes detected"
    else
        log_warn "Changes detected! To commit and push:"
        echo ""
        echo -e "${YELLOW}  cd $GITHUB_PAGES_DIR${NC}"
        echo -e "${YELLOW}  git add docs/manual/${NC}"
        echo -e "${YELLOW}  git commit -m \"Update user manual - $TIMESTAMP\"${NC}"
        echo -e "${YELLOW}  git push${NC}"
        echo ""
    fi

    log_success "Manual sync completed successfully!"
    log_info "Log file: $LOG_FILE"
}

# Create log file if it doesn't exist
touch "$LOG_FILE"

# Run main function
main "$@"

exit 0

# Legacy index.html generation (keeping for reference)
cat > /dev/null << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HelixCode User Manual</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
        }
        .links {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }
        .link-card {
            padding: 20px;
            background: #f9f9f9;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            text-decoration: none;
            color: #333;
            transition: all 0.3s;
        }
        .link-card:hover {
            background: #e8f4f8;
            border-color: #4a90e2;
            transform: translateY(-2px);
        }
        .link-title {
            font-weight: 600;
            font-size: 1.1em;
            margin-bottom: 5px;
        }
        .link-desc {
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>HelixCode User Manual</h1>
        <p class="subtitle">Comprehensive documentation for the HelixCode platform</p>

        <div class="links">
            <a href="USER_MANUAL.html" class="link-card">
                <div class="link-title">HTML Version</div>
                <div class="link-desc">Read the manual in your browser with formatted styling</div>
            </a>

            <a href="USER_MANUAL.md" class="link-card">
                <div class="link-title">Markdown Version</div>
                <div class="link-desc">Raw Markdown file for offline reading or conversion</div>
            </a>
        </div>
    </div>
</body>
</html>
EOF

log_success "Created index.html"

# Create basic CSS for the manual
cat > "$MANUAL_DEST_DIR/style.css" << 'EOF'
:root {
    --primary-color: #4a90e2;
    --text-color: #333;
    --bg-color: #ffffff;
    --border-color: #e0e0e0;
    --code-bg: #f5f5f5;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    line-height: 1.6;
    color: var(--text-color);
    max-width: 900px;
    margin: 0 auto;
    padding: 20px;
    background: #f9f9f9;
}

#TOC {
    background: var(--bg-color);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 20px;
    margin-bottom: 30px;
}

#TOC ul {
    list-style-type: none;
    padding-left: 20px;
}

#TOC > ul {
    padding-left: 0;
}

#TOC a {
    color: var(--primary-color);
    text-decoration: none;
}

#TOC a:hover {
    text-decoration: underline;
}

h1, h2, h3, h4, h5, h6 {
    margin-top: 1.5em;
    margin-bottom: 0.5em;
    font-weight: 600;
}

h1 {
    border-bottom: 3px solid var(--primary-color);
    padding-bottom: 10px;
}

h2 {
    border-bottom: 2px solid var(--border-color);
    padding-bottom: 5px;
}

code {
    background: var(--code-bg);
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 0.9em;
}

pre {
    background: var(--code-bg);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 15px;
    overflow-x: auto;
}

pre code {
    background: none;
    padding: 0;
}

table {
    border-collapse: collapse;
    width: 100%;
    margin: 20px 0;
}

th, td {
    border: 1px solid var(--border-color);
    padding: 10px;
    text-align: left;
}

th {
    background: var(--code-bg);
    font-weight: 600;
}

blockquote {
    border-left: 4px solid var(--primary-color);
    margin-left: 0;
    padding-left: 20px;
    color: #666;
}

a {
    color: var(--primary-color);
}

hr {
    border: none;
    border-top: 2px solid var(--border-color);
    margin: 30px 0;
}
EOF

log_success "Created style.css"

# Generate sync metadata
cat > "$MANUAL_DEST_DIR/.sync-metadata.json" << EOF
{
  "last_sync": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "source": "$MANUAL_SRC",
  "version": "$(grep -m1 'Version' "$MANUAL_SRC" | sed 's/.*Version \([0-9.]*\).*/\1/' || echo 'unknown')",
  "sync_script": "$0"
}
EOF

log_success "Generated sync metadata"

# Summary
echo ""
log_success "Manual sync completed successfully!"
echo ""
log_info "Generated files:"
echo "  - $MANUAL_DEST_DIR/USER_MANUAL.md"
echo "  - $MANUAL_DEST_DIR/index.html"
echo "  - $MANUAL_DEST_DIR/style.css"
if [ -f "$MANUAL_DEST_DIR/USER_MANUAL.html" ]; then
    echo "  - $MANUAL_DEST_DIR/USER_MANUAL.html"
fi
echo ""
log_info "View at: file://$MANUAL_DEST_DIR/index.html"
echo ""
