#!/bin/bash

# HelixCode Local LLM - Comprehensive Test Suite Runner
# This script runs all test types with proper configuration and reporting

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
TEST_RESULTS_DIR="$PROJECT_ROOT/test-results"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
PARALLEL_JOBS=${PARALLEL_JOBS:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}
TEST_TIMEOUT=${TEST_TIMEOUT:-10m}

# Create results directory
mkdir -p "$TEST_RESULTS_DIR"

# Logging
log() {
    echo -e "${CYAN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

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

# Header
header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║             HelixCode Local LLM Test Suite              ║"
    echo "║              Comprehensive Testing Framework                ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Pre-flight checks
preflight_checks() {
    log "🔍 Running pre-flight checks..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go version: $GO_VERSION"
    
    # Check required dependencies
    local deps=("git" "curl")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Required dependency not found: $dep"
            exit 1
        fi
        log_success "$dep is available"
    done
    
    # Check test dependencies
    log "Checking test dependencies..."
    if ! go list -m github.com/stretchr/testify &> /dev/null; then
        log_warning "Installing test dependencies..."
        go get github.com/stretchr/testify@latest
    fi
    
    log_success "✅ Pre-flight checks passed"
}

# Build test runner
build_test_runner() {
    log "🔨 Building test runner..."
    
    cd "$PROJECT_ROOT"
    if ! go build -tags=test_runner -o test_runner test_runner.go; then
        log_error "Failed to build test runner"
        exit 1
    fi
    
    log_success "✅ Test runner built successfully"
}

# Build CLI for tests
build_cli() {
    log "🔨 Building CLI for tests..."
    
    cd "$PROJECT_ROOT"
    if ! go build -tags=test -o local-llm-test local-llm-test.go; then
        log_error "Failed to build CLI"
        exit 1
    fi
    
    # Test CLI basic functionality
    if ! ./local-llm-test --help &> /dev/null; then
        log_error "CLI built but basic functionality failed"
        exit 1
    fi
    
    log_success "✅ CLI built successfully"
}

# Run specific test suite
run_test_suite() {
    local suite_name=$1
    local suite_args=$2
    
    log "🧪 Running $suite_name test suite..."
    
    local log_file="$TEST_RESULTS_DIR/${suite_name,,}-$TIMESTAMP.log"
    local start_time=$(date +%s)
    
    # Run the test suite
    if ./test_runner $suite_args > "$log_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_success "$suite_name tests passed in ${duration}s"
        log "📝 Results saved to: $log_file"
        
        # Copy to latest
        cp "$log_file" "$TEST_RESULTS_DIR/${suite_name,,}-latest.log"
        
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_error "$suite_name tests failed after ${duration}s"
        log "📝 Error log: $log_file"
        
        # Show last 20 lines of error
        log_error "Last 20 lines of error:"
        tail -20 "$log_file" | sed 's/^/    /'
        
        # Copy to latest
        cp "$log_file" "$TEST_RESULTS_DIR/${suite_name,,}-latest.log"
        
        return 1
    fi
}

# Security tests
run_security_tests() {
    log_info "Running security and compliance tests..."
    
    # SonarQube-like static analysis
    log_info "📊 Running static code analysis..."
    cd "$PROJECT_ROOT"
    
    # Run go vet
    if ! go vet ./...; then
        log_error "Static analysis found issues"
        return 1
    fi
    
    # Run gosec for security scanning
    if command -v gosec &> /dev/null; then
        log_info "🔒 Running security scanner..."
        gosec ./... 2>&1 | tee "$TEST_RESULTS_DIR/security-scan-$TIMESTAMP.log"
    else
        log_warning "gosec not found, skipping security scan"
    fi
    
    # Run security tests
    run_test_suite "Security" "-security"
}

# Unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    run_test_suite "Unit" "-unit -timeout $TEST_TIMEOUT"
}

# Integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    run_test_suite "Integration" "-integration -timeout $((TEST_TIMEOUT * 2))"
}

# End-to-end tests
run_e2e_tests() {
    log_info "Running end-to-end tests..."
    run_test_suite "E2E" "-e2e -timeout $((TEST_TIMEOUT * 3))"
}

# Hardware automation tests
run_automation_tests() {
    log_info "Running hardware automation tests..."
    run_test_suite "Automation" "-automation -timeout $((TEST_TIMEOUT * 4))"
}

# Performance benchmarks
run_benchmarks() {
    log_info "📈 Running performance benchmarks..."
    
    cd "$PROJECT_ROOT"
    local benchmark_log="$TEST_RESULTS_DIR/benchmarks-$TIMESTAMP.log"
    
    if go test -bench=. -benchmem ./... 2>&1 | tee "$benchmark_log"; then
        log_success "Benchmarks completed successfully"
    else
        log_error "Benchmarks failed"
        return 1
    fi
}

# Race condition tests
run_race_tests() {
    log_info "🏃 Running race condition tests..."
    
    cd "$PROJECT_ROOT"
    local race_log="$TEST_RESULTS_DIR/race-tests-$TIMESTAMP.log"
    
    if go test -race -short ./... 2>&1 | tee "$race_log"; then
        log_success "Race condition tests passed"
    else
        log_error "Race condition tests failed"
        return 1
    fi
}

# Memory leak tests
run_memory_tests() {
    log_info "💾 Running memory leak tests..."
    
    cd "$PROJECT_ROOT"
    local memory_log="$TEST_RESULTS_DIR/memory-tests-$TIMESTAMP.log"
    
    # Run tests with memory profiling
    if GODEBUG=gctrace=1 go test -memprofile=mem.prof ./... 2>&1 | tee "$memory_log"; then
        log_success "Memory tests completed"
        
        # Analyze memory profile if available
        if command -v go &> /dev/null && [[ -f mem.prof ]]; then
            go tool pprof -text mem.prof > "$TEST_RESULTS_DIR/memory-profile-$TIMESTAMP.txt"
            log_success "Memory profile generated"
        fi
    else
        log_error "Memory tests failed"
        return 1
    fi
}

# Coverage report
generate_coverage() {
    log_info "📊 Generating coverage report..."
    
    cd "$PROJECT_ROOT"
    local coverage_file="$TEST_RESULTS_DIR/coverage-$TIMESTAMP.out"
    local coverage_html="$TEST_RESULTS_DIR/coverage-$TIMESTAMP.html"
    
    # Run tests with coverage
    if go test -coverprofile="$coverage_file" -covermode=atomic ./...; then
        log_success "Coverage data generated"
        
        # Generate HTML report
        if go tool cover -html="$coverage_file" -o "$coverage_html"; then
            log_success "Coverage HTML report generated: $coverage_html"
        fi
        
        # Show coverage summary
        go tool cover -func="$coverage_file" > "$TEST_RESULTS_DIR/coverage-summary-$TIMESTAMP.txt"
        
        # Extract total coverage percentage
        local total_coverage=$(tail -1 "$TEST_RESULTS_DIR/coverage-summary-$TIMESTAMP.txt" | grep -o '[0-9.]*%' || echo "N/A")
        log_info "Total coverage: $total_coverage"
        
        # Copy to latest
        cp "$coverage_file" "$TEST_RESULTS_DIR/coverage-latest.out"
        cp "$coverage_html" "$TEST_RESULTS_DIR/coverage-latest.html"
        
    else
        log_error "Failed to generate coverage"
        return 1
    fi
}

# Test report generation
generate_test_report() {
    log_info "📋 Generating comprehensive test report..."
    
    local report_file="$TEST_RESULTS_DIR/test-report-$TIMESTAMP.md"
    
    cat > "$report_file" << EOF
# HelixCode Local LLM - Test Report

**Generated:** $(date)  
**Test Duration:** ${TOTAL_DURATION:-0}s  

## Summary

| Test Suite | Status | Duration | Details |
|-------------|--------|----------|---------|
EOF
    
    # Add suite results
    local total_passed=0
    local total_failed=0
    local total_skipped=0
    
    for suite_log in "$TEST_RESULTS_DIR"/*-latest.log; do
        if [[ -f "$suite_log" ]]; then
            local suite_name=$(basename "$suite_log" "-latest.log" | tr '[:lower:]' '[:upper:]')
            local status="❌ FAILED"
            
            if grep -q "ALL TESTS PASSED" "$suite_log" 2>/dev/null; then
                status="✅ PASSED"
            fi
            
            # Extract counts from log
            local passed=$(grep -o "PASSED: [0-9]*" "$suite_log" 2>/dev/null | grep -o "[0-9]*" | head -1 || echo "0")
            local failed=$(grep -o "FAILED: [0-9]*" "$suite_log" 2>/dev/null | grep -o "[0-9]*" | head -1 || echo "0")
            local skipped=$(grep -o "SKIPPED: [0-9]*" "$suite_log" 2>/dev/null | grep -o "[0-9]*" | head -1 || echo "0")
            
            echo "| $suite_name | $status | - | Passed: $passed, Failed: $failed, Skipped: $skipped |" >> "$report_file"
            
            total_passed=$((total_passed + passed))
            total_failed=$((total_failed + failed))
            total_skipped=$((total_skipped + skipped))
        fi
    done
    
    cat >> "$report_file" << EOF
| **TOTAL** | - | - | **Passed: $total_passed, Failed: $total_failed, Skipped: $total_skipped** |

## Coverage

EOF
    
    # Add coverage info
    if [[ -f "$TEST_RESULTS_DIR/coverage-summary-$TIMESTAMP.txt" ]]; then
        echo '```' >> "$report_file"
        cat "$TEST_RESULTS_DIR/coverage-summary-$TIMESTAMP.txt" >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF

## Hardware Information

\`\`\`
EOF
    
    # Add hardware info
    uname -a >> "$report_file"
    echo "" >> "$report_file"
    
    if command -v lscpu &> /dev/null; then
        echo "CPU Info:" >> "$report_file"
        lscpu | grep -E "(Model name|CPU\(s\)|Thread)" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if command -v free &> /dev/null; then
        echo "Memory Info:" >> "$report_file"
        free -h >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if command -v nvidia-smi &> /dev/null; then
        echo "GPU Info:" >> "$report_file"
        nvidia-smi >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF
\`\`\`

## Recommendations

EOF
    
    # Add recommendations based on results
    if [[ $total_failed -gt 0 ]]; then
        echo "- ❌ Some tests failed. Review the detailed logs for each test suite." >> "$report_file"
    fi
    
    if [[ $total_skipped -gt 0 ]]; then
        echo "- ⚠️ Some tests were skipped. Consider running without --skip-expensive flags for complete testing." >> "$report_file"
    fi
    
    log_success "Test report generated: $report_file"
}

# Cleanup function
cleanup() {
    log_info "🧹 Cleaning up..."
    
    # Remove temporary files
    rm -f "$PROJECT_ROOT/mem.prof"
    rm -f "$PROJECT_ROOT/test_runner"
    rm -f "$PROJECT_ROOT/local-llm-test"
    
    # Compress old logs (keep last 5)
    find "$TEST_RESULTS_DIR" -name "*.log" -type f -mtime +7 -exec gzip {} \;
    find "$TEST_RESULTS_DIR" -name "*.html" -type f -mtime +7 -exec gzip {} \;
    
    log_success "✅ Cleanup completed"
}

# Main execution
main() {
    # Parse command line arguments
    local run_all=true
    local run_security=false
    local run_unit=false
    local run_integration=false
    local run_e2e=false
    local run_automation=false
    local run_benchmarks=false
    local run_coverage=false
    local skip_expensive=false
    local skip_hardware=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --all)
                run_all=true
                shift
                ;;
            --security)
                run_security=true
                run_all=false
                shift
                ;;
            --unit)
                run_unit=true
                run_all=false
                shift
                ;;
            --integration)
                run_integration=true
                run_all=false
                shift
                ;;
            --e2e)
                run_e2e=true
                run_all=false
                shift
                ;;
            --automation)
                run_automation=true
                run_all=false
                shift
                ;;
            --benchmarks)
                run_benchmarks=true
                run_all=false
                shift
                ;;
            --coverage)
                run_coverage=true
                run_all=false
                shift
                ;;
            --skip-expensive)
                skip_expensive=true
                export SKIP_EXPENSIVE_TESTS=true
                shift
                ;;
            --skip-hardware)
                skip_hardware=true
                export SKIP_HARDWARE_TESTS=true
                shift
                ;;
            --parallel=*)
                PARALLEL_JOBS="${1#*=}"
                shift
                ;;
            --timeout=*)
                TEST_TIMEOUT="${1#*=}"
                shift
                ;;
            --preflight)
                # Pre-flight checks
                log "🔍 Running pre-flight checks..."
                
                if ! command -v go &> /dev/null; then
                    log_error "Go is not installed or not in PATH"
                    exit 1
                fi
                
                GO_VERSION=$(go version | awk '{print $3}')
                log_success "Go version: $GO_VERSION"
                
                # Check required dependencies
                local deps=("git" "curl")
                for dep in "${deps[@]}"; do
                    if ! command -v "$dep" &> /dev/null; then
                        log_error "Required dependency not found: $dep"
                        exit 1
                    fi
                    log_success "$dep is available"
                done
                
                # Check test dependencies
                log "Checking test dependencies..."
                if ! go list -m github.com/stretchr/testify &> /dev/null; then
                    log_warning "Installing test dependencies..."
                    go get github.com/stretchr/testify@latest
                fi
                
                log_success "✅ All pre-flight checks passed"
                exit 0
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --all              Run all test suites (default)"
                echo "  --security         Run security and compliance tests"
                echo "  --unit             Run unit tests"
                echo "  --integration      Run integration tests"
                echo "  --e2e              Run end-to-end tests"
                echo "  --automation       Run hardware automation tests"
                echo "  --benchmarks       Run performance benchmarks"
                echo "  --coverage         Generate coverage report"
                echo "  --skip-expensive   Skip expensive/time-consuming tests"
                echo "  --skip-hardware    Skip hardware-dependent tests"
                echo "  --parallel=N       Run N tests in parallel"
                echo "  --timeout=D        Test timeout duration"
                echo "  --help, -h         Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Set traps for cleanup
    trap cleanup EXIT
    
    # Start execution
    header
    
    local overall_start_time=$(date +%s)
    local failed_suites=0
    
    # Pre-flight checks
    preflight_checks
    
    # Build components
    build_test_runner
    build_cli
    
    # Export environment variables
    export PARALLEL_JOBS=$PARALLEL_JOBS
    export TEST_TIMEOUT=$TEST_TIMEOUT
    
    # Run selected test suites
    if [[ $run_all == true ]] || [[ $run_security == true ]]; then
        if ! run_security_tests; then
            ((failed_suites++))
        fi
    fi
    
    if [[ $run_all == true ]] || [[ $run_unit == true ]]; then
        if ! run_unit_tests; then
            ((failed_suites++))
        fi
    fi
    
    if [[ $run_all == true ]] || [[ $run_integration == true ]]; then
        if ! run_integration_tests; then
            ((failed_suites++))
        fi
    fi
    
    if [[ $run_all == true ]] || [[ $run_e2e == true ]]; then
        if ! run_e2e_tests; then
            ((failed_suites++))
        fi
    fi
    
    if [[ $run_all == true ]] || [[ $run_automation == true ]]; then
        if ! run_automation_tests; then
            ((failed_suites++))
        fi
    fi
    
    if [[ $run_all == true ]] || [[ $run_benchmarks == true ]]; then
        run_benchmarks || true # Don't fail on benchmarks
    fi
    
    if [[ $run_all == true ]] || [[ $run_coverage == true ]]; then
        generate_coverage || true # Don't fail on coverage
    fi
    
    # Additional tests
    if [[ $run_all == true ]]; then
        run_race_tests || true
        run_memory_tests || true
    fi
    
    # Calculate total duration
    local overall_end_time=$(date +%s)
    export TOTAL_DURATION=$((overall_end_time - overall_start_time))
    
    # Generate final report
    generate_test_report
    
    # Final result
    echo -e "\n${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                      FINAL RESULT                           ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    
    if [[ $failed_suites -eq 0 ]]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED SUCCESSFULLY! 🎉${NC}"
        echo -e "${GREEN}✅ Total duration: ${TOTAL_DURATION}s${NC}"
        echo -e "${GREEN}📁 Results saved to: $TEST_RESULTS_DIR${NC}"
        exit 0
    else
        echo -e "${RED}❌ $failed_suites test suite(s) failed${NC}"
        echo -e "${RED}⏱️  Total duration: ${TOTAL_DURATION}s${NC}"
        echo -e "${RED}📁 Results saved to: $TEST_RESULTS_DIR${NC}"
        echo -e "${RED}🔍 Check the log files for detailed error information${NC}"
        exit 1
    fi
}

# Execute main function
main "$@"