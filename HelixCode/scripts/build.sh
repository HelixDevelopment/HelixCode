#!/bin/bash
#
# build.sh - Comprehensive build script for HelixCode
#
# This script handles the complete build process including:
# - Code compilation
# - Asset generation
# - Documentation sync
# - Testing
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

# Build configuration
BUILD_DOCS=true
RUN_TESTS=false
SKIP_ASSETS=false
VERBOSE=false
PLATFORM="all"

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

Comprehensive build script for HelixCode.

Options:
    --no-docs           Skip documentation generation
    --no-assets         Skip asset generation
    --with-tests        Run tests after build
    --platform PLATFORM Build for specific platform (linux|darwin|windows|all)
    -v, --verbose       Verbose output
    -h, --help          Show this help message

Examples:
    $0                          # Standard build
    $0 --with-tests             # Build and test
    $0 --no-docs --with-tests   # Skip docs, build and test
    $0 --platform linux         # Build for Linux only

EOF
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-docs)
            BUILD_DOCS=false
            shift
            ;;
        --no-assets)
            SKIP_ASSETS=false
            shift
            ;;
        --with-tests)
            RUN_TESTS=true
            shift
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
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
echo -e "${CYAN}║       HelixCode Build Script v1.0          ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════╝${NC}"
echo ""

START_TIME=$(date +%s)
log_info "Build started at: $(date +"%Y-%m-%d %H:%M:%S")"
log_info "Platform: $PLATFORM"
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Step 1: Check prerequisites
log_step "1/6 Checking prerequisites..."

if ! command -v go &> /dev/null; then
    log_error "Go is not installed"
    exit 1
fi
log_success "Go $(go version | awk '{print $3}')"

if [ "$BUILD_DOCS" = true ]; then
    if ! command -v pandoc &> /dev/null; then
        log_warn "pandoc not found - documentation conversion will be limited"
    else
        log_success "pandoc $(pandoc --version | head -n1 | awk '{print $2}')"
    fi
fi

echo ""

# Step 2: Generate assets
if [ "$SKIP_ASSETS" = false ]; then
    log_step "2/6 Generating assets..."

    # Logo assets
    if [ -f "$SCRIPT_DIR/logo/generate_assets.go" ]; then
        cd "$SCRIPT_DIR/logo"
        go run generate_assets.go
        cd "$PROJECT_ROOT"
        log_success "Logo assets generated"
    else
        log_warn "Logo generation script not found"
    fi

    echo ""
else
    log_info "2/6 Skipping asset generation"
    echo ""
fi

# Step 3: Build documentation
if [ "$BUILD_DOCS" = true ]; then
    log_step "3/6 Building documentation..."

    # Sync manual
    if [ -x "$SCRIPT_DIR/sync-manual.sh" ]; then
        "$SCRIPT_DIR/sync-manual.sh"
    else
        log_warn "sync-manual.sh not found or not executable"
    fi

    echo ""
else
    log_info "3/6 Skipping documentation build"
    echo ""
fi

# Step 4: Build application
log_step "4/6 Building application..."

# Get build metadata
VERSION="1.0.0"
BUILD_TIME=$(date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

log_info "Version: $VERSION"
log_info "Build Time: $BUILD_TIME"
log_info "Git Commit: $GIT_COMMIT"
echo ""

LDFLAGS="-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT"

case $PLATFORM in
    linux)
        log_info "Building for Linux..."
        GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-linux ./cmd/server
        log_success "Built bin/helixcode-linux"
        ;;
    darwin)
        log_info "Building for macOS..."
        GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-macos ./cmd/server
        log_success "Built bin/helixcode-macos"
        ;;
    windows)
        log_info "Building for Windows..."
        GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-windows.exe ./cmd/server
        log_success "Built bin/helixcode-windows.exe"
        ;;
    all)
        log_info "Building for all platforms..."

        # Current platform
        go build -ldflags="$LDFLAGS" -o bin/helixcode ./cmd/server
        log_success "Built bin/helixcode (native)"

        # Linux
        GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-linux ./cmd/server
        log_success "Built bin/helixcode-linux"

        # macOS
        GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-macos ./cmd/server
        log_success "Built bin/helixcode-macos"

        # Windows
        GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/helixcode-windows.exe ./cmd/server
        log_success "Built bin/helixcode-windows.exe"
        ;;
    *)
        log_error "Unknown platform: $PLATFORM"
        exit 1
        ;;
esac

echo ""

# Step 5: Run tests (optional)
if [ "$RUN_TESTS" = true ]; then
    log_step "5/6 Running tests..."

    if go test -v ./... 2>&1 | tee /tmp/helixcode-test.log; then
        log_success "All tests passed"
    else
        log_error "Tests failed - see /tmp/helixcode-test.log"
        exit 1
    fi

    echo ""
else
    log_info "5/6 Skipping tests (use --with-tests to enable)"
    echo ""
fi

# Step 6: Generate build info
log_step "6/6 Generating build information..."

BUILD_INFO_FILE="bin/BUILD_INFO.json"
mkdir -p bin

cat > "$BUILD_INFO_FILE" << EOF
{
  "version": "$VERSION",
  "build_time": "$BUILD_TIME",
  "git_commit": "$GIT_COMMIT",
  "platform": "$PLATFORM",
  "go_version": "$(go version | awk '{print $3}')",
  "docs_built": $BUILD_DOCS,
  "tests_run": $RUN_TESTS,
  "built_by": "$(whoami)@$(hostname)",
  "build_script": "$0"
}
EOF

log_success "Build information saved to $BUILD_INFO_FILE"

echo ""

# Calculate build time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo -e "${CYAN}╔════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║           Build Complete                   ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════╝${NC}"
echo ""

log_success "Build completed successfully in ${DURATION}s"
echo ""
log_info "Build artifacts:"
ls -lh bin/ | grep -v total | awk '{print "  - " $9 " (" $5 ")"}'
echo ""

if [ "$BUILD_DOCS" = true ]; then
    log_info "Documentation available at: website/manual/index.html"
fi

log_info "Build info: $BUILD_INFO_FILE"
echo ""
