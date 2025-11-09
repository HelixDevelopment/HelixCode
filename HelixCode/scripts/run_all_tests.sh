#!/bin/bash

# =============================================================================
# HelixCode API Key Management - Test Execution Script
# =============================================================================
# This script runs all API key management tests with comprehensive reporting
# =============================================================================

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$PROJECT_ROOT/tests"
COVERAGE_FILE="$PROJECT_ROOT/coverage.out"
HTML_COVERAGE="$PROJECT_ROOT/coverage.html"
TEST_RESULTS="$PROJECT_ROOT/test_results.txt"
PERFORMANCE_RESULTS="$PROJECT_ROOT/performance_results.txt"

# Timestamp
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}=== $1 ===${NC}"
}

# Test results
UNIT_TESTS_PASSED=false
INTEGRATION_TESTS_PASSED=false
PERFORMANCE_TESTS_PASSED=false
OVERALL_PASSED=false

# Performance metrics
TOTAL_UNIT_TESTS=0
PASSED_UNIT_TESTS=0
TOTAL_INTEGRATION_TESTS=0
PASSED_INTEGRATION_TESTS=0
TOTAL_PERFORMANCE_TESTS=0
PASSED_PERFORMANCE_TESTS=0

# =============================================================================
# PRE-TEST SETUP
# =============================================================================

setup_test_environment() {
    log_header "Setting Up Test Environment"
    
    # Create results directory
    mkdir -p "$PROJECT_ROOT/results"
    
    # Clean up previous results
    rm -f "$COVERAGE_FILE" "$HTML_COVERAGE" "$TEST_RESULTS" "$PERFORMANCE_RESULTS"
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check project structure
    if [[ ! -d "$TEST_DIR" ]]; then
        log_error "Test directory not found: $TEST_DIR"
        exit 1
    fi
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    cd "$PROJECT_ROOT"
    go mod download
    go mod tidy
    
    # Check for test files
    if ! find "$TEST_DIR" -name "*_test.go" | grep -q .; then
        log_error "No test files found in test directory"
        exit 1
    fi
    
    log_success "Test environment setup completed"
}

# =============================================================================
# UNIT TESTS
# =============================================================================

run_unit_tests() {
    log_header "Running Unit Tests"
    
    local unit_test_dir="$TEST_DIR/unit"
    
    if [[ ! -d "$unit_test_dir" ]]; then
        log_warning "Unit test directory not found, skipping..."
        return 0
    fi
    
    log_info "Executing unit tests with coverage..."
    
    # Run unit tests with coverage
    cd "$PROJECT_ROOT"
    if go test -v -race -coverprofile=unit_coverage.out -covermode=atomic \
        "./tests/unit" 2>&1 | tee "$PROJECT_ROOT/results/unit_tests.txt"; then
        UNIT_TESTS_PASSED=true
        log_success "Unit tests passed"
    else
        log_error "Unit tests failed"
        UNIT_TESTS_PASSED=false
    fi
    
    # Count unit tests
    if [[ -f "$PROJECT_ROOT/results/unit_tests.txt" ]]; then
        TOTAL_UNIT_TESTS=$(grep -c "^--- RUN:" "$PROJECT_ROOT/results/unit_tests.txt" || echo "0")
        PASSED_UNIT_TESTS=$(grep -c "^--- PASS:" "$PROJECT_ROOT/results/unit_tests.txt" || echo "0")
    fi
    
    log_info "Unit Tests: $PASSED_UNIT_TESTS/$TOTAL_UNIT_TESTS passed"
}

# =============================================================================
# INTEGRATION TESTS
# =============================================================================

run_integration_tests() {
    log_header "Running Integration Tests"
    
    local integration_test_dir="$TEST_DIR/integration"
    
    if [[ ! -d "$integration_test_dir" ]]; then
        log_warning "Integration test directory not found, skipping..."
        return 0
    fi
    
    log_info "Executing integration tests with coverage..."
    
    # Run integration tests with coverage
    cd "$PROJECT_ROOT"
    if go test -v -race -coverprofile=integration_coverage.out -covermode=atomic \
        "./tests/integration" 2>&1 | tee "$PROJECT_ROOT/results/integration_tests.txt"; then
        INTEGRATION_TESTS_PASSED=true
        log_success "Integration tests passed"
    else
        log_error "Integration tests failed"
        INTEGRATION_TESTS_PASSED=false
    fi
    
    # Count integration tests
    if [[ -f "$PROJECT_ROOT/results/integration_tests.txt" ]]; then
        TOTAL_INTEGRATION_TESTS=$(grep -c "^--- RUN:" "$PROJECT_ROOT/results/integration_tests.txt" || echo "0")
        PASSED_INTEGRATION_TESTS=$(grep -c "^--- PASS:" "$PROJECT_ROOT/results/integration_tests.txt" || echo "0")
    fi
    
    log_info "Integration Tests: $PASSED_INTEGRATION_TESTS/$TOTAL_INTEGRATION_TESTS passed"
}

# =============================================================================
# PERFORMANCE TESTS
# =============================================================================

run_performance_tests() {
    log_header "Running Performance Tests"
    
    local performance_test_dir="$TEST_DIR/performance"
    
    if [[ ! -d "$performance_test_dir" ]]; then
        log_warning "Performance test directory not found, skipping..."
        return 0
    fi
    
    log_info "Executing performance tests with benchmarks..."
    
    # Run performance tests with benchmarks
    cd "$PROJECT_ROOT"
    if go test -v -bench=. -benchmem -run=^$ \
        "./tests/performance" 2>&1 | tee "$PROJECT_ROOT/results/performance_tests.txt"; then
        PERFORMANCE_TESTS_PASSED=true
        log_success "Performance tests passed"
    else
        log_error "Performance tests failed"
        PERFORMANCE_TESTS_PASSED=false
    fi
    
    # Count performance tests
    if [[ -f "$PROJECT_ROOT/results/performance_tests.txt" ]]; then
        TOTAL_PERFORMANCE_TESTS=$(grep -c "^--- RUN:" "$PROJECT_ROOT/results/performance_tests.txt" || echo "0")
        PASSED_PERFORMANCE_TESTS=$(grep -c "^--- PASS:" "$PROJECT_ROOT/results/performance_tests.txt" || echo "0")
    fi
    
    log_info "Performance Tests: $PASSED_PERFORMANCE_TESTS/$TOTAL_PERFORMANCE_TESTS passed"
}

# =============================================================================
# COVERAGE REPORTING
# =============================================================================

generate_coverage_report() {
    log_header "Generating Coverage Report"
    
    cd "$PROJECT_ROOT"
    
    # Combine coverage files
    local coverage_files=()
    [[ -f "unit_coverage.out" ]] && coverage_files+=("unit_coverage.out")
    [[ -f "integration_coverage.out" ]] && coverage_files+=("integration_coverage.out")
    
    if [[ ${#coverage_files[@]} -gt 0 ]]; then
        log_info "Combining coverage reports..."
        
        # Install gocovmerge if not present
        if ! command -v gocovmerge &> /dev/null; then
            log_info "Installing gocovmerge..."
            go install github.com/wadey/gocovmerge@latest
        fi
        
        # Combine coverage files
        gocovmerge "${coverage_files[@]}" > "$COVERAGE_FILE"
        
        # Generate HTML coverage report
        log_info "Generating HTML coverage report..."
        go tool cover -html="$COVERAGE_FILE" -o "$HTML_COVERAGE"
        
        # Show coverage percentage
        local coverage_percent=$(go tool cover -func="$COVERAGE_FILE" | grep "total:" | awk '{print $3}')
        log_info "Total Coverage: $coverage_percent"
        
        # Coverage threshold check
        local coverage_number=$(echo "$coverage_percent" | sed 's/%//')
        if (( $(echo "$coverage_number >= 95" | bc -l) )); then
            log_success "Coverage meets threshold (>=95%)"
        else
            log_warning "Coverage below threshold (<95%)"
        fi
    else
        log_warning "No coverage files found to combine"
    fi
}

# =============================================================================
# PERFORMANCE ANALYSIS
# =============================================================================

analyze_performance() {
    log_header "Analyzing Performance Results"
    
    local perf_file="$PROJECT_ROOT/results/performance_tests.txt"
    
    if [[ -f "$perf_file" ]]; then
        log_info "Performance analysis results:"
        
        # Extract benchmark results
        if grep -q "^Benchmark" "$perf_file"; then
            echo "Benchmark Results:" >> "$PERFORMANCE_RESULTS"
            grep "^Benchmark" "$perf_file" | while read -r line; do
                local bench_name=$(echo "$line" | awk '{print $1}')
                local iterations=$(echo "$line" | awk '{print $2}')
                local ns_per_op=$(echo "$line" | awk '{print $3}')
                local bytes_per_op=$(echo "$line" | awk '{print $4}')
                local allocs_per_op=$(echo "$line" | awk '{print $5}')
                
                echo "  $bench_name: $iterations iterations, $ns_per_op ns/op, $bytes_per_op bytes/op, $allocs_per_op allocs/op" >> "$PERFORMANCE_RESULTS"
            done
        fi
        
        # Performance summary
        local total_benchmarks=$(grep -c "^Benchmark" "$perf_file" || echo "0")
        log_info "Total Benchmarks: $total_benchmarks"
        
        # Check for performance regressions
        if grep -q "FAIL" "$perf_file"; then
            log_error "Performance regressions detected"
            return 1
        else
            log_success "No performance regressions detected"
        fi
    else
        log_warning "No performance test results found"
    fi
}

# =============================================================================
# TEST REPORT GENERATION
# =============================================================================

generate_test_report() {
    log_header "Generating Test Report"
    
    local report_file="$PROJECT_ROOT/results/test_report.md"
    
    cat > "$report_file" << EOF
# API Key Management Test Report

## Test Execution Summary

**Timestamp:** $TIMESTAMP  
**Project Root:** $PROJECT_ROOT  

### Test Results

| Test Suite | Total Tests | Passed | Status | Coverage |
|------------|------------|---------|---------|----------|
| Unit Tests | $TOTAL_UNIT_TESTS | $PASSED_UNIT_TESTS | $([ "$UNIT_TESTS_PASSED" = true ] && echo "✅ PASSED" || echo "❌ FAILED") | $(get_coverage_percentage unit_coverage.out) |
| Integration Tests | $TOTAL_INTEGRATION_TESTS | $PASSED_INTEGRATION_TESTS | $([ "$INTEGRATION_TESTS_PASSED" = true ] && echo "✅ PASSED" || echo "❌ FAILED") | $(get_coverage_percentage integration_coverage.out) |
| Performance Tests | $TOTAL_PERFORMANCE_TESTS | $PASSED_PERFORMANCE_TESTS | $([ "$PERFORMANCE_TESTS_PASSED" = true ] && echo "✅ PASSED" || echo "❌ FAILED") | N/A |

### Overall Status

$([ "$UNIT_TESTS_PASSED" = true ] && [ "$INTEGRATION_TESTS_PASSED" = true ] && [ "$PERFORMANCE_TESTS_PASSED" = true ] && echo "✅ ALL TESTS PASSED" || echo "❌ SOME TESTS FAILED")

### Coverage Summary

**Total Coverage:** $(go tool cover -func="$COVERAGE_FILE" 2>/dev/null | grep "total:" | awk '{print $3}' || echo "N/A")

### Performance Summary

$(cat "$PERFORMANCE_RESULTS" 2>/dev/null || echo "No performance data available")

### Detailed Results

- [Unit Test Results]($(pwd)/results/unit_tests.txt)
- [Integration Test Results]($(pwd)/results/integration_tests.txt)
- [Performance Test Results]($(pwd)/results/performance_tests.txt)
- [HTML Coverage Report]($(pwd)/coverage.html)

EOF
    
    log_success "Test report generated: $report_file"
}

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

get_coverage_percentage() {
    local coverage_file="$1"
    
    if [[ -f "$coverage_file" ]]; then
        local coverage=$(go tool cover -func="$coverage_file" 2>/dev/null | grep "total:" | awk '{print $3}')
        echo "${coverage:-N/A}"
    else
        echo "N/A"
    fi
}

check_overall_status() {
    if [[ "$UNIT_TESTS_PASSED" = true && "$INTEGRATION_TESTS_PASSED" = true && "$PERFORMANCE_TESTS_PASSED" = true ]]; then
        OVERALL_PASSED=true
        log_success "All tests passed successfully!"
        return 0
    else
        OVERALL_PASSED=false
        log_error "Some tests failed!"
        return 1
    fi
}

# =============================================================================
# CLEANUP
# =============================================================================

cleanup() {
    log_info "Cleaning up temporary files..."
    
    # Remove temporary coverage files
    rm -f unit_coverage.out integration_coverage.out
    
    # Compress old test results if needed
    if [[ -d "$PROJECT_ROOT/results" ]]; then
        find "$PROJECT_ROOT/results" -name "*.txt" -mtime +7 -exec gzip {} \; 2>/dev/null || true
    fi
    
    log_info "Cleanup completed"
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

main() {
    log_header "HelixCode API Key Management Test Suite"
    log_info "Starting test execution at: $TIMESTAMP"
    
    # Set up error handling
    trap cleanup EXIT
    trap 'log_error "Test execution interrupted"; exit 1' INT TERM
    
    # Run setup
    setup_test_environment
    
    # Run tests
    run_unit_tests
    run_integration_tests
    run_performance_tests
    
    # Generate reports
    generate_coverage_report
    analyze_performance
    generate_test_report
    
    # Check overall status
    if check_overall_status; then
        log_success "=== ALL TESTS COMPLETED SUCCESSFULLY ==="
        exit_code=0
    else
        log_error "=== SOME TESTS FAILED ==="
        exit_code=1
    fi
    
    # Display summary
    log_header "Test Execution Summary"
    echo "Unit Tests:       $PASSED_UNIT_TESTS/$TOTAL_UNIT_TESTS passed"
    echo "Integration Tests: $PASSED_INTEGRATION_TESTS/$TOTAL_INTEGRATION_TESTS passed"
    echo "Performance Tests: $PASSED_PERFORMANCE_TESTS/$TOTAL_PERFORMANCE_TESTS passed"
    echo "Overall Status:    $([ "$OVERALL_PASSED" = true ] && echo "✅ PASSED" || echo "❌ FAILED")"
    echo "Report:           $PROJECT_ROOT/results/test_report.html"
    
    exit $exit_code
}

# Run main function
main "$@"