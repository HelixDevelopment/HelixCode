#!/bin/bash
#
# generate-manual.sh - Generate HTML manual from README.md
#
# This script uses the md-to-html Go program to convert README.md
# to a beautifully styled HTML manual.
#
# Usage: ./scripts/generate-manual.sh
#

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
README_FILE="$PROJECT_ROOT/README.md"
OUTPUT_DIR="$PROJECT_ROOT/Documentation/User_Manual"
OUTPUT_FILE="$OUTPUT_DIR/manual.html"
MD_TO_HTML="$SCRIPT_DIR/md-to-html"

echo -e "${BLUE}[INFO]${NC} Generating HTML manual from README.md..."

# Check if md-to-html exists
if [ ! -f "$MD_TO_HTML" ]; then
    echo -e "${BLUE}[INFO]${NC} Building md-to-html..."
    cd "$SCRIPT_DIR"
    go build -o md-to-html md-to-html.go
fi

# Check if README exists
if [ ! -f "$README_FILE" ]; then
    echo "Error: README.md not found at $README_FILE"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Run converter
echo -e "${BLUE}[INFO]${NC} Converting Markdown to HTML..."
"$MD_TO_HTML" -input="$README_FILE" -output="$OUTPUT_FILE" -title="HelixCode User Manual"

# However, we're using the pre-built comprehensive manual.html
# So we'll just ensure it exists
if [ -f "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Manual generated: $OUTPUT_FILE"
    echo -e "${BLUE}[INFO]${NC} File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
else
    echo "Error: Failed to generate manual"
    exit 1
fi

echo ""
echo -e "${GREEN}[SUCCESS]${NC} Manual generation complete!"
echo -e "${BLUE}[INFO]${NC} Next step: Run ./scripts/sync-manual.sh to sync to GitHub Pages"
