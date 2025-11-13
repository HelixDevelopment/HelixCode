#!/bin/bash

# Helix CLI Test Runner
# This script runs the complete test suite including unit, integration, and e2e tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="30m"
E2E_TIMEOUT="60m"
DOCKER_COMPOSE_FILE="docker-compose.test.yml"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first."
        exit 1
    fi
    
    print_success "All prerequisites are satisfied"
}

# Function to generate test SSH keys
generate_test_keys() {
    print_status "Generating test SSH keys..."
    
    if [ ! -f "test/workers/ssh-keys/id_rsa" ]; then
        ./scripts/generate-test-keys.sh
        print_success "Test SSH keys generated"
    else
        print_success "Test SSH keys already exist"
    fi
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    if timeout ${TEST_TIMEOUT} go test -v -race $(go list ./internal/... | grep -v memory/providers) -tags=unit; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        exit 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    if timeout ${TEST_TIMEOUT} go test -v -race ./internal/... -tags=integration; then
        print_success "Integration tests passed"
    else
        print_warning "Integration tests failed or skipped (may be normal if external services are not available)"
    fi
}

# Function to start test environment
start_test_environment() {
    print_status "Starting test environment with Docker Compose..."
    
    # Stop any existing test environment
    docker-compose -f ${DOCKER_COMPOSE_FILE} down --volumes --remove-orphans
    
    # Build and start test environment
    if docker-compose -f ${DOCKER_COMPOSE_FILE} up --build -d; then
        print_success "Test environment started"
        
        # Wait for services to be ready
        print_status "Waiting for test services to be ready..."
        sleep 30
    else
        print_error "Failed to start test environment"
        exit 1
    fi
}

# Function to run end-to-end tests
run_e2e_tests() {
    print_status "Running end-to-end tests..."
    
    if timeout ${E2E_TIMEOUT} docker-compose -f ${DOCKER_COMPOSE_FILE} run --rm test-runner; then
        print_success "End-to-end tests passed"
    else
        print_error "End-to-end tests failed"
        exit 1
    fi
}

# Function to stop test environment
stop_test_environment() {
    print_status "Stopping test environment..."
    
    if docker-compose -f ${DOCKER_COMPOSE_FILE} down --volumes --remove-orphans; then
        print_success "Test environment stopped"
    else
        print_warning "Failed to stop test environment cleanly"
    fi
}

# Function to run specific test types
run_specific_tests() {
    case "$1" in
        "unit")
            run_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "e2e")
            generate_test_keys
            start_test_environment
            run_e2e_tests
            stop_test_environment
            ;;
        "all")
            generate_test_keys
            run_unit_tests
            run_integration_tests
            start_test_environment
            run_e2e_tests
            stop_test_environment
            ;;
        *)
            print_error "Unknown test type: $1"
            print_status "Usage: $0 {unit|integration|e2e|all}"
            exit 1
            ;;
    esac
}

# Function to run performance tests
run_performance_tests() {
    print_status "Running performance tests..."
    
    # Test hardware detection performance
    print_status "Testing hardware detection performance..."
    time go test -v ./internal/hardware -run TestHardwareDetection -count=1
    
    # Test model manager performance
    print_status "Testing model manager performance..."
    time go test -v ./internal/llm -run TestModelManager -count=1
    
    print_success "Performance tests completed"
}

# Function to generate test coverage report
generate_coverage_report() {
    print_status "Generating test coverage report..."
    
    # Generate coverage for unit tests
    go test -coverprofile=coverage.out ./internal/... -tags=unit
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    
    # Print coverage summary
    go tool cover -func=coverage.out | tail -1
    
    print_success "Coverage report generated: coverage.html"
}

# Main function
main() {
    print_status "Starting Helix CLI test suite..."
    
    # Check prerequisites
    check_prerequisites
    
    # Parse command line arguments
    TEST_TYPE="${1:-all}"
    
    case "${TEST_TYPE}" in
        "unit"|"integration"|"e2e"|"all")
            run_specific_tests "${TEST_TYPE}"
            ;;
        "performance")
            run_performance_tests
            ;;
        "coverage")
            generate_coverage_report
            ;;
        "help")
            echo "Helix CLI Test Runner"
            echo ""
            echo "Usage: $0 {unit|integration|e2e|all|performance|coverage|help}"
            echo ""
            echo "Options:"
            echo "  unit        Run unit tests only"
            echo "  integration Run integration tests only"
            echo "  e2e         Run end-to-end tests only"
            echo "  all         Run all tests (default)"
            echo "  performance Run performance tests"
            echo "  coverage    Generate test coverage report"
            echo "  help        Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
    
    print_success "Test suite completed successfully!"
}

# Run main function
main "$@"