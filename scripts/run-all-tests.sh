#!/bin/bash
set -e

# HelixCode Comprehensive Test Runner
# Runs all tests across the entire codebase

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    ((FAILED_TESTS++))
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((PASSED_TESTS++))
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if we're in the right directory
check_environment() {
    if [ ! -f "${PROJECT_ROOT}/HelixCode/go.mod" ]; then
        error "Not in HelixCode project root directory"
        exit 1
    fi
}

# Run tests for a specific package
run_package_tests() {
    local package_path="$1"
    local package_name="$2"

    log "Running tests for $package_name..."

    if [ -d "$package_path" ]; then
        cd "$PROJECT_ROOT/$package_path"

        if go test -v ./... -timeout 60s; then
            success "$package_name tests passed"
        else
            error "$package_name tests failed"
        fi
    else
        warning "Package directory $package_path not found, skipping"
    fi
}

# Run all unit tests
run_unit_tests() {
    log "Running all unit tests..."

    cd "$PROJECT_ROOT/HelixCode"

    # Run tests for all packages
    run_package_tests "." "main"
    run_package_tests "internal/auth" "auth"
    run_package_tests "internal/config" "config"
    run_package_tests "internal/database" "database"
    run_package_tests "internal/hardware" "hardware"
    run_package_tests "internal/llm" "llm"
    run_package_tests "internal/mcp" "mcp"
    run_package_tests "internal/notification" "notification"
    run_package_tests "internal/project" "project"
    run_package_tests "internal/redis" "redis"
    run_package_tests "internal/server" "server"
    run_package_tests "internal/session" "session"
    run_package_tests "internal/task" "task"
    run_package_tests "internal/worker" "worker"
    run_package_tests "shared/mobile-core" "mobile-core"

    # Run tests for applications
    run_package_tests "applications/terminal-ui" "terminal-ui"
    run_package_tests "applications/desktop" "desktop"
    run_package_tests "applications/aurora-os" "aurora-os"
    run_package_tests "applications/symphony-os" "symphony-os"
}

# Run integration tests
run_integration_tests() {
    log "Running integration tests..."

    cd "$PROJECT_ROOT/HelixCode"

    # Run integration tests (marked with Integration in test name)
    if go test -v ./... -run Integration -timeout 120s; then
        success "Integration tests passed"
    else
        error "Integration tests failed"
    fi
}

# Run end-to-end tests
run_e2e_tests() {
    log "Running end-to-end tests..."

    cd "$PROJECT_ROOT/HelixCode"

    if [ -d "test/e2e" ]; then
        if go test -v ./test/e2e/... -timeout 300s; then
            success "E2E tests passed"
        else
            error "E2E tests failed"
        fi
    else
        warning "E2E test directory not found, skipping"
    fi
}

# Generate coverage report
run_coverage() {
    log "Generating comprehensive coverage report..."

    cd "$PROJECT_ROOT/HelixCode"

    # Run coverage for all packages
    go test -coverprofile="$PROJECT_ROOT/coverage.out" -covermode=atomic ./...

    # Run coverage for task package specifically
    go test -coverprofile="$PROJECT_ROOT/task-coverage.out" -covermode=atomic ./internal/task/...

    # Display coverage summary
    if [ -f "$PROJECT_ROOT/coverage.out" ]; then
        log "Overall coverage summary:"
        go tool cover -func="$PROJECT_ROOT/coverage.out" | tail -1
    fi

    if [ -f "$PROJECT_ROOT/task-coverage.out" ]; then
        log "Task package coverage summary:"
        go tool cover -func="$PROJECT_ROOT/task-coverage.out" | tail -1
    fi

    success "Coverage report generated"
}

# Run linting
run_linting() {
    log "Running linting checks..."

    cd "$PROJECT_ROOT/HelixCode"

    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run ./...; then
            success "Linting passed"
        else
            error "Linting failed"
        fi
    else
        warning "golangci-lint not found, skipping linting"
    fi
}

# Run build checks
run_build_checks() {
    log "Running build checks..."

    cd "$PROJECT_ROOT/HelixCode"

    # Try to build all applications
    applications=("terminal-ui" "desktop" "aurora-os" "symphony-os" "server")

    for app in "${applications[@]}"; do
        log "Building $app..."
        if go build -o "/tmp/$app" "./cmd/$app" 2>/dev/null; then
            success "$app build successful"
        else
            error "$app build failed"
        fi
    done
}

# Main execution
main() {
    local test_type="${1:-all}"

    log "Starting HelixCode comprehensive test suite"
    log "Test type: $test_type"

    check_environment

    case "$test_type" in
        "unit")
            run_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "e2e")
            run_e2e_tests
            ;;
        "coverage")
            run_coverage
            ;;
        "lint")
            run_linting
            ;;
        "build")
            run_build_checks
            ;;
        "all")
            run_linting
            run_build_checks
            run_unit_tests
            run_integration_tests
            run_e2e_tests
            run_coverage
            ;;
        *)
            error "Unknown test type: $test_type"
            echo "Available test types: unit, integration, e2e, coverage, lint, build, all"
            exit 1
            ;;
    esac

    # Print summary
    log "Test Summary:"
    log "  Total: $((PASSED_TESTS + FAILED_TESTS))"
    log "  Passed: $PASSED_TESTS"
    log "  Failed: $FAILED_TESTS"

    if [ $FAILED_TESTS -eq 0 ]; then
        success "All tests completed successfully!"
        exit 0
    else
        error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Run main function
main "$@"