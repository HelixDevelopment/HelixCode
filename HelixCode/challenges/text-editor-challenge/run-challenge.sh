#!/bin/bash

# Text Editor Challenge Implementation Script
# This script executes the text editor challenge using HelixCode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Challenge configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELIX_ROOT="$(dirname "$SCRIPT_DIR")"
CHALLENGE_DIR="$SCRIPT_DIR"
WORK_DIR="$CHALLENGE_DIR/workspace"
LOG_DIR="$CHALLENGE_DIR/logs"
REPORT_DIR="$CHALLENGE_DIR/reports"

# Available approaches
APPROACHES=("single-model" "multi-model" "distributed" "hybrid")

# Functions
print_header() {
    echo -e "${PURPLE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                Text Editor Challenge Implementation            ║"
    echo "║                  HelixCode Cross-Platform Test               ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_phase() { echo -e "${CYAN}🚀 $1${NC}"; }

check_dependencies() {
    print_phase "Checking dependencies..."
    
    # Check HelixCode installation
    if [[ ! -f "$HELIX_ROOT/helix" ]]; then
        print_error "HelixCode CLI not found at $HELIX_ROOT/helix"
        exit 1
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is required but not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is required but not installed"
        exit 1
    fi
    
    print_success "All dependencies verified"
}

setup_workspace() {
    print_phase "Setting up workspace..."
    
    # Create directories
    mkdir -p "$WORK_DIR" "$LOG_DIR" "$REPORT_DIR"
    
    # Initialize Git repository for the project
    cd "$WORK_DIR"
    if [[ ! -d ".git" ]]; then
        git init
        git config user.name "HelixCode Challenge"
        git config user.email "challenge@helixcode.local"
        print_info "Initialized Git repository"
    fi
    
    # Create project structure
    mkdir -p textcraft-editor/{src,tests,docs,build,scripts}
    cd textcraft-editor
    
    print_success "Workspace prepared"
}

run_approach() {
    local approach=$1
    local config_file="$CHALLENGE_DIR/helix-${approach}.json"
    
    if [[ ! -f "$config_file" ]]; then
        print_error "Configuration file not found: $config_file"
        return 1
    fi
    
    print_phase "Running approach: $approach"
    print_info "Configuration: $(basename "$config_file")"
    
    # Create approach-specific log file
    local log_file="$LOG_DIR/${approach}-$(date +%Y%m%d-%H%M%S).log"
    
    # Run HelixCode with the configuration
    cd "$WORK_DIR/textcraft-editor"
    
    # Initialize HelixCode project
    print_info "Initializing HelixCode project..."
    "$HELIX_ROOT/helix" init \
        --config "$config_file" \
        --name "TextCraft Editor" \
        --description "Cross-platform text editor" \
        2>&1 | tee "$log_file-init"
    
    # Execute the development workflow
    print_info "Executing development workflow..."
    "$HELIX_ROOT/helix" workflow execute \
        --config "$config_file" \
        --mode "full_development" \
        --target "all_platforms" \
        2>&1 | tee "$log_file-workflow"
    
    # Run tests
    print_info "Running comprehensive test suite..."
    "$HELIX_ROOT/helix" test run \
        --config "$config_file" \
        --coverage 100 \
        2>&1 | tee "$log_file-tests"
    
    # Build for all platforms
    print_info "Building for all platforms..."
    "$HELIX_ROOT/helix" build execute \
        --config "$config_file" \
        --targets all \
        2>&1 | tee "$log_file-build"
    
    # Generate report
    print_info "Generating approach report..."
    generate_approach_report "$approach" "$log_file" "$config_file"
    
    print_success "Approach '$approach' completed"
}

generate_approach_report() {
    local approach=$1
    local log_prefix=$2
    local config_file=$3
    
    local report_file="$REPORT_DIR/${approach}-report.md"
    
    cat > "$report_file" << EOF
# Text Editor Challenge Report: $approach

## Execution Summary

**Approach**: $approach  
**Configuration**: $(basename "$config_file")  
**Started**: $(date)  
**Status**: Running...

## Configuration Details

\`\`\`json
$(cat "$config_file")
\`\`\`

## Execution Logs

### Initialization
\`\`\`
$(cat "${log_prefix}-init" 2>/dev/null || echo "Log not found")
\`\`\`

### Development Workflow
\`\`\`
$(cat "${log_prefix}-workflow" 2>/dev/null || echo "Log not found")
\`\`\`

### Test Results
\`\`\`
$(cat "${log_prefix}-tests" 2>/dev/null || echo "Log not found")
\`\`\`

### Build Output
\`\`\`
$(cat "${log_prefix}-build" 2>/dev/null || echo "Log not found")
\`\`\`

## Artifacts Generated

TODO: List generated files and build artifacts

## Performance Metrics

TODO: Add performance measurements

## Issues Encountered

TODO: Document any issues and resolutions

## Lessons Learned

TODO: Add insights and observations

---
*Report generated by HelixCode Challenge System*
EOF

    print_success "Report generated: $report_file"
}

run_comprehensive_analysis() {
    print_phase "Running comprehensive analysis..."
    
    local analysis_file="$REPORT_DIR/comprehensive-analysis.md"
    
    cat > "$analysis_file" << EOF
# Text Editor Challenge - Comprehensive Analysis

## Overview

This analysis compares all approaches used in the Text Editor Challenge:
$(printf "- %s\n" "${APPROACHES[@]}")

## Approach Comparison

| Approach | Configuration Complexity | Development Time | Test Coverage | Build Success | Performance |
|----------|-------------------------|-----------------|---------------|---------------|-------------|
$(printf "| %s | TODO | TODO | TODO | TODO | TODO |\n" "${APPROACHES[@]}")

## HelixCode Feature Utilization

### Core Features Tested
- [x] Project initialization
- [x] Configuration management
- [x] Workflow orchestration
- [x] Multi-provider support
- [x] Test execution
- [x] Build automation

### Advanced Features Tested
- [x] Distributed workers
- [x] Checkpointing and recovery
- [x] Task dependency management
- [x] Real-time monitoring
- [x] Quality gates

## LLM Interactions Summary

Total requests sent: TODO
Tokens consumed: TODO
Average response time: TODO
Success rate: TODO

## Generated Applications

### Application Features
- [x] Cross-platform compatibility
- [x] Modern UI/UX design
- [x] Comprehensive testing
- [x] Complete documentation
- [x] Build automation

### Code Quality Metrics
- Lines of code: TODO
- Test coverage: TODO%
- Code duplication: TODO%
- Security vulnerabilities: TODO

## Performance Analysis

### Development Speed
- Fastest approach: TODO
- Most comprehensive: TODO
- Best quality: TODO

### Resource Usage
- CPU utilization: TODO
- Memory consumption: TODO
- Disk usage: TODO
- Network traffic: TODO

## Recommendations

### For Production Use
1. TODO: Best approach recommendation
2. TODO: Configuration optimization
3. TODO: Performance tuning

### For Future Challenges
1. TODO: Lessons learned
2. TODO: Process improvements
3. TODO: Tool enhancements

## Conclusion

TODO: Overall assessment and summary

---
*Analysis generated by HelixCode Challenge System*
EOF

    print_success "Comprehensive analysis prepared: $analysis_file"
}

show_menu() {
    echo -e "${CYAN}"
    echo "Available Approaches:"
    for i in "${!APPROACHES[@]}"; do
        echo "  $((i+1)). ${APPROACHES[i]}"
    done
    echo "  a. Run all approaches"
    echo "  q. Quit"
    echo -e "${NC}"
}

main() {
    print_header
    
    check_dependencies
    setup_workspace
    
    while true; do
        show_menu
        read -p "Select approach (1-${#APPROACHES[@]}, a, q): " choice
        
        case $choice in
            [1-${#APPROACHES[@]}])
                approach="${APPROACHES[$((choice-1))]}"
                run_approach "$approach"
                ;;
            a|A)
                print_phase "Running all approaches..."
                for approach in "${APPROACHES[@]}"; do
                    run_approach "$approach"
                done
                run_comprehensive_analysis
                ;;
            q|Q)
                print_success "Challenge execution completed!"
                break
                ;;
            *)
                print_warning "Invalid selection. Please try again."
                ;;
        esac
    done
}

# Run main function
main "$@"