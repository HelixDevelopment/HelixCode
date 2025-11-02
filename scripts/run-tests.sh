#!/bin/bash
set -e

# HelixCode Test Runner Script
# Runs various types of tests with coverage support

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TEST_TYPE="${1:-all}"
COVERAGE_FILE="${PROJECT_ROOT}/coverage.out"
TASK_COVERAGE_FILE="${PROJECT_ROOT}/task-coverage.out"

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if we're in the right directory
check_environment() {
    if [ ! -f "${PROJECT_ROOT}/go.mod" ]; then
        error "Not in HelixCode project root directory"
        exit 1
    fi
}

# Run unit tests
run_unit_tests() {
    log "Running unit tests..."

    cd "$PROJECT_ROOT"
    go test -v ./... -short -timeout 30s

    success "Unit tests completed"
}

# Run integration tests
run_integration_tests() {
    log "Running integration tests..."

    cd "$PROJECT_ROOT"
    go test -v ./... -run Integration -timeout 60s

    success "Integration tests completed"
}

# Run end-to-end tests
run_e2e_tests() {
    log "Running end-to-end tests..."

    cd "$PROJECT_ROOT"

    if [ -d "test/e2e" ]; then
        go test -v ./test/e2e/... -timeout 120s
    else
        warning "E2E test directory not found, skipping"
    fi

    success "E2E tests completed"
}

# Run distributed tests
run_distributed_tests() {
    log "Running distributed tests..."

    cd "$PROJECT_ROOT"
    go test -v ./internal/worker/... -run Distributed -timeout 120s

    success "Distributed tests completed"
}

# Run all tests
run_all_tests() {
    log "Running all tests..."

    run_unit_tests
    run_integration_tests
    run_distributed_tests
    run_e2e_tests

    success "All tests completed"
}

# Generate coverage report
run_coverage() {
    log "Running tests with coverage..."

    cd "$PROJECT_ROOT"

    # Run coverage for main packages
    go test -coverprofile="$COVERAGE_FILE" -covermode=atomic ./... -short

    # Run coverage for task-related packages specifically
    go test -coverprofile="$TASK_COVERAGE_FILE" -covermode=atomic ./internal/task/...

    # Display coverage summary
    if [ -f "$COVERAGE_FILE" ]; then
        log "Coverage summary:"
        go tool cover -func="$COVERAGE_FILE" | tail -1
    fi

    if [ -f "$TASK_COVERAGE_FILE" ]; then
        log "Task package coverage summary:"
        go tool cover -func="$TASK_COVERAGE_FILE" | tail -1
    fi

    success "Coverage report generated"
}

# Show usage
usage() {
    echo "Usage: $0 [test_type]"
    echo ""
    echo "Test types:"
    echo "  unit         - Run unit tests only"
    echo "  integration  - Run integration tests only"
    echo "  distributed  - Run distributed worker tests only"
    echo "  e2e          - Run end-to-end tests only"
    echo "  coverage     - Run tests with coverage report"
    echo "  all          - Run all test types (default)"
}

# Main execution
main() {
    local test_type="$1"

    if [ "$test_type" = "help" ] || [ "$test_type" = "--help" ] || [ "$test_type" = "-h" ]; then
        usage
        exit 0
    fi

    log "Starting HelixCode test suite"
    log "Test type: ${test_type:-all}"

    check_environment

    case "$test_type" in
        "unit")
            run_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "distributed")
            run_distributed_tests
            ;;
        "e2e")
            run_e2e_tests
            ;;
        "coverage")
            run_coverage
            ;;
        "all")
            run_all_tests
            ;;
        *)
            error "Unknown test type: $test_type"
            usage
            exit 1
            ;;
    esac

    success "Test execution completed successfully"
}

# Run main function
main "$@"