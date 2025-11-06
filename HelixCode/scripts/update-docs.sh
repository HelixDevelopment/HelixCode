#!/bin/bash
#
# update-docs.sh - Master script for updating all documentation
#
# This script is the single command for updating all project documentation:
# - Converts Markdown to HTML
# - Syncs to website
# - Updates timestamps
# - Generates changelog entries
# - Optionally generates PDF
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCS_DIR="$PROJECT_ROOT/docs"
WEBSITE_DIR="$PROJECT_ROOT/website"

# Configuration
GENERATE_PDF=false
UPDATE_CHANGELOG=false
VERBOSE=false

# Function to print colored messages
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Master script for updating all project documentation.

Options:
    -p, --pdf           Generate PDF version of the manual
    -c, --changelog     Update changelog with documentation changes
    -v, --verbose       Verbose output
    -h, --help          Show this help message

Examples:
    $0                      # Basic update (Markdown to HTML + sync)
    $0 --pdf                # Include PDF generation
    $0 --pdf --changelog    # Full update with PDF and changelog

EOF
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--pdf)
            GENERATE_PDF=true
            shift
            ;;
        -c|--changelog)
            UPDATE_CHANGELOG=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            ;;
    esac
done

# Print banner
echo ""
echo -e "${CYAN}╔════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   HelixCode Documentation Update Tool     ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════╝${NC}"
echo ""

CURRENT_DATE=$(date +"%Y-%m-%d %H:%M:%S")
log_info "Started at: $CURRENT_DATE"
echo ""

# Step 1: Update timestamps
log_step "1/6 Updating timestamps in Markdown files..."

MARKDOWN_FILES=(
    "$DOCS_DIR/USER_MANUAL.md"
    "$DOCS_DIR/USER_GUIDE.md"
    "$DOCS_DIR/API_REFERENCE.md"
    "$DOCS_DIR/ARCHITECTURE.md"
)

TIMESTAMP=$(date +%Y-%m-%d)

for file in "${MARKDOWN_FILES[@]}"; do
    if [ -f "$file" ]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            sed -i '' "s/Last Updated: .*/Last Updated: $TIMESTAMP/" "$file"
        else
            # Linux
            sed -i "s/Last Updated: .*/Last Updated: $TIMESTAMP/" "$file"
        fi
        log_success "Updated timestamp: $(basename "$file")"
    elif [ "$VERBOSE" = true ]; then
        log_warn "File not found: $(basename "$file")"
    fi
done

echo ""

# Step 2: Convert Markdown to HTML
log_step "2/6 Converting Markdown to HTML..."

if command -v pandoc &> /dev/null; then
    mkdir -p "$WEBSITE_DIR/manual"
    mkdir -p "$WEBSITE_DIR/guides"

    # Convert USER_MANUAL.md
    if [ -f "$DOCS_DIR/USER_MANUAL.md" ]; then
        pandoc "$DOCS_DIR/USER_MANUAL.md" \
            -f markdown \
            -t html5 \
            --standalone \
            --toc \
            --toc-depth=3 \
            --css=style.css \
            --metadata title="HelixCode User Manual" \
            -o "$WEBSITE_DIR/manual/USER_MANUAL.html"
        log_success "Generated USER_MANUAL.html"
    fi

    # Convert USER_GUIDE.md
    if [ -f "$DOCS_DIR/USER_GUIDE.md" ]; then
        pandoc "$DOCS_DIR/USER_GUIDE.md" \
            -f markdown \
            -t html5 \
            --standalone \
            --toc \
            --css=../manual/style.css \
            --metadata title="HelixCode User Guide" \
            -o "$WEBSITE_DIR/guides/USER_GUIDE.html"
        log_success "Generated USER_GUIDE.html"
    fi

    # Convert API_REFERENCE.md
    if [ -f "$DOCS_DIR/API_REFERENCE.md" ]; then
        pandoc "$DOCS_DIR/API_REFERENCE.md" \
            -f markdown \
            -t html5 \
            --standalone \
            --toc \
            --css=../manual/style.css \
            --metadata title="HelixCode API Reference" \
            -o "$WEBSITE_DIR/guides/API_REFERENCE.html"
        log_success "Generated API_REFERENCE.html"
    fi
else
    log_error "pandoc not found - cannot convert to HTML"
    log_info "Install with: brew install pandoc (macOS) or apt-get install pandoc (Linux)"
    exit 1
fi

echo ""

# Step 3: Sync to website
log_step "3/6 Syncing documentation to website..."

if [ -x "$SCRIPT_DIR/sync-manual.sh" ]; then
    "$SCRIPT_DIR/sync-manual.sh"
else
    log_error "sync-manual.sh not found or not executable"
    exit 1
fi

echo ""

# Step 4: Generate PDF (optional)
if [ "$GENERATE_PDF" = true ]; then
    log_step "4/6 Generating PDF versions..."

    if command -v pandoc &> /dev/null; then
        mkdir -p "$WEBSITE_DIR/pdf"

        # Generate PDF for USER_MANUAL
        if [ -f "$DOCS_DIR/USER_MANUAL.md" ]; then
            pandoc "$DOCS_DIR/USER_MANUAL.md" \
                -f markdown \
                -t pdf \
                --pdf-engine=xelatex \
                --toc \
                --toc-depth=3 \
                --metadata title="HelixCode User Manual" \
                --metadata author="HelixCode Team" \
                --metadata date="$TIMESTAMP" \
                -o "$WEBSITE_DIR/pdf/HelixCode_User_Manual.pdf"
            log_success "Generated HelixCode_User_Manual.pdf"
        fi
    else
        log_warn "pandoc not found - skipping PDF generation"
    fi
else
    log_info "4/6 Skipping PDF generation (use --pdf to enable)"
fi

echo ""

# Step 5: Update changelog (optional)
if [ "$UPDATE_CHANGELOG" = true ]; then
    log_step "5/6 Updating CHANGELOG.md..."

    CHANGELOG="$PROJECT_ROOT/CHANGELOG.md"

    if [ -f "$CHANGELOG" ]; then
        # Create temporary file with new entry
        TEMP_CHANGELOG=$(mktemp)

        cat > "$TEMP_CHANGELOG" << EOF
# Changelog

## [Unreleased]

### Documentation
- Updated documentation on $TIMESTAMP
- Regenerated HTML and PDF versions
- Synced manual to website

EOF

        # Append rest of changelog (skip first two lines if they're header)
        tail -n +3 "$CHANGELOG" >> "$TEMP_CHANGELOG"

        # Replace original changelog
        mv "$TEMP_CHANGELOG" "$CHANGELOG"

        log_success "Updated CHANGELOG.md"
    else
        log_warn "CHANGELOG.md not found - creating new one"

        cat > "$CHANGELOG" << EOF
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Documentation
- Updated documentation on $TIMESTAMP
- Regenerated HTML and PDF versions
- Synced manual to website

EOF
        log_success "Created CHANGELOG.md"
    fi
else
    log_info "5/6 Skipping changelog update (use --changelog to enable)"
fi

echo ""

# Step 6: Generate metadata
log_step "6/6 Generating documentation metadata..."

cat > "$WEBSITE_DIR/.doc-metadata.json" << EOF
{
  "generated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "version": "$(grep -m1 'Version' "$DOCS_DIR/USER_MANUAL.md" | sed 's/.*Version \([0-9.]*\).*/\1/' || echo 'unknown')",
  "files_updated": [
    "USER_MANUAL.md",
    "USER_GUIDE.md",
    "API_REFERENCE.md"
  ],
  "formats": {
    "html": true,
    "markdown": true,
    "pdf": $GENERATE_PDF
  },
  "tools": {
    "pandoc": "$(command -v pandoc &> /dev/null && pandoc --version | head -n1 || echo 'not installed')",
    "script_version": "1.0.0"
  }
}
EOF

log_success "Generated documentation metadata"

echo ""
echo -e "${CYAN}╔════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║         Documentation Update Complete      ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════╝${NC}"
echo ""

# Summary
log_success "Documentation updated successfully!"
echo ""
log_info "Summary:"
echo "  ✓ Timestamps updated"
echo "  ✓ HTML files generated"
echo "  ✓ Manual synced to website"

if [ "$GENERATE_PDF" = true ]; then
    echo "  ✓ PDF files generated"
fi

if [ "$UPDATE_CHANGELOG" = true ]; then
    echo "  ✓ Changelog updated"
fi

echo ""
log_info "Documentation available at:"
echo "  - Website: $WEBSITE_DIR"
echo "  - Manual: file://$WEBSITE_DIR/manual/index.html"

if [ "$GENERATE_PDF" = true ]; then
    echo "  - PDF: $WEBSITE_DIR/pdf/"
fi

echo ""
log_info "Next steps:"
echo "  1. Review the generated documentation"
echo "  2. Commit changes: git add docs/ website/"
echo "  3. Push to repository: git push"
echo ""
