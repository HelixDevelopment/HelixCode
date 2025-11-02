#!/bin/bash
set -e

# HelixCode Distributed Testing Script
# Runs comprehensive tests across multiple worker nodes

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
COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.test.yml"

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

# Check if docker and docker-compose are available
check_dependencies() {
    log "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    success "Dependencies check passed"
}

# Start the test infrastructure
start_infrastructure() {
    log "Starting test infrastructure..."

    cd "$PROJECT_ROOT"

    # Stop any existing containers
    docker-compose -f "$COMPOSE_FILE" down --volumes --remove-orphans 2>/dev/null || true

    # Start services
    docker-compose -f "$COMPOSE_FILE" up -d

    # Wait for services to be healthy
    log "Waiting for services to be ready..."

    # Wait for PostgreSQL
    log "Waiting for PostgreSQL..."
    timeout=60
    while [ $timeout -gt 0 ]; do
        if docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U helixcode -d helixcode_test >/dev/null 2>&1; then
            success "PostgreSQL is ready"
            break
        fi
        sleep 2
        timeout=$((timeout - 2))
    done

    if [ $timeout -le 0 ]; then
        error "PostgreSQL failed to start"
        docker-compose -f "$COMPOSE_FILE" logs postgres
        exit 1
    fi

    # Wait for Redis
    log "Waiting for Redis..."
    timeout=30
    while [ $timeout -gt 0 ]; do
        if docker-compose -f "$COMPOSE_FILE" exec -T redis redis-cli ping | grep -q PONG; then
            success "Redis is ready"
            break
        fi
        sleep 2
        timeout=$((timeout - 2))
    done

    if [ $timeout -le 0 ]; then
        error "Redis failed to start"
        docker-compose -f "$COMPOSE_FILE" logs redis
        exit 1
    fi

    # Wait for workers
    log "Waiting for workers to register..."
    sleep 10

    success "Test infrastructure started successfully"
}

# Run tests based on type
run_tests() {
    local test_type="$1"

    case "$test_type" in
        "unit")
            log "Running unit tests..."
            docker-compose -f "$COMPOSE_FILE" exec -T test-runner go test -v ./... -short
            ;;
        "integration")
            log "Running integration tests..."
            docker-compose -f "$COMPOSE_FILE" exec -T test-runner go test -v ./... -run Integration
            ;;
        "distributed")
            log "Running distributed tests..."
            docker-compose -f "$COMPOSE_FILE" exec -T test-runner go test -v ./internal/worker/... -run Distributed
            ;;
        "e2e")
            log "Running end-to-end tests..."
            docker-compose -f "$COMPOSE_FILE" exec -T test-runner go test -v ./test/e2e/...
            ;;
        "coverage")
            log "Running tests with coverage..."
            docker-compose -f "$COMPOSE_FILE" exec -T test-runner ./scripts/run-tests.sh coverage
            ;;
        "all")
            log "Running all tests..."
            run_tests "unit"
            run_tests "integration"
            run_tests "distributed"
            run_tests "e2e"
            ;;
        *)
            error "Unknown test type: $test_type"
            echo "Available test types: unit, integration, distributed, e2e, coverage, all"
            exit 1
            ;;
    esac
}

# Generate coverage report
generate_coverage_report() {
    log "Generating coverage report..."

    # Run coverage tests
    run_tests "coverage"

    # Copy coverage files from container
    docker-compose -f "$COMPOSE_FILE" exec -T test-runner cat coverage.out > "${PROJECT_ROOT}/coverage.out" 2>/dev/null || true
    docker-compose -f "$COMPOSE_FILE" exec -T test-runner cat task-coverage.out > "${PROJECT_ROOT}/task-coverage.out" 2>/dev/null || true

    if [ -f "${PROJECT_ROOT}/coverage.out" ]; then
        success "Coverage report generated: coverage.out"
        go tool cover -func=coverage.out | tail -1
    fi
}

# Clean up test infrastructure
cleanup() {
    log "Cleaning up test infrastructure..."

    cd "$PROJECT_ROOT"
    docker-compose -f "$COMPOSE_FILE" down --volumes --remove-orphans

    success "Cleanup completed"
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
    echo ""
    echo "Environment variables:"
    echo "  COMPOSE_FILE - Path to docker-compose file (default: docker-compose.test.yml)"
}

# Main execution
main() {
    local test_type="$1"

    if [ "$test_type" = "help" ] || [ "$test_type" = "--help" ] || [ "$test_type" = "-h" ]; then
        usage
        exit 0
    fi

    log "Starting HelixCode distributed testing suite"
    log "Test type: ${test_type:-all}"

    check_dependencies
    start_infrastructure

    # Run tests
    run_tests "${test_type:-all}"

    # Generate coverage if requested
    if [ "$test_type" = "coverage" ]; then
        generate_coverage_report
    fi

    success "All tests completed successfully"

    # Cleanup (optional - comment out if you want to inspect containers)
    # cleanup
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"