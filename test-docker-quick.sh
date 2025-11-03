#!/bin/bash

# Quick HelixCode Docker Setup Test
# Tests core functionality without full builds

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    print_info "Running test: $test_name"
    
    if eval "$test_command"; then
        print_success "Test passed: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        print_error "Test failed: $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
}

main() {
    echo ""
    echo "🚀 HelixCode Docker Quick Test"
    echo "=============================="
    echo ""
    
    # Basic file checks
    run_test "Dockerfile exists" "[ -f Dockerfile ]"
    run_test "docker-compose.helix.yml exists" "[ -f docker-compose.helix.yml ]"
    run_test "helix script exists and executable" "[ -x helix ]"
    run_test "docker-entrypoint.sh exists and executable" "[ -x docker-entrypoint.sh ]"
    run_test "test-docker-setup.sh exists and executable" "[ -x test-docker-setup.sh ]"
    
    # Configuration checks
    run_test ".env file exists" "[ -f .env ]"
    run_test ".env.example exists" "[ -f .env.example ]"
    run_test "DOCKER_SETUP.md exists" "[ -f DOCKER_SETUP.md ]"
    run_test "README_DOCKER.md exists" "[ -f README_DOCKER.md ]"
    
    # Script functionality
    run_test "helix script help works" "./helix help >/dev/null 2>&1"
    run_test "helix status works" "./helix status >/dev/null 2>&1"
    
    # Directory structure
    run_test "HelixCode source directory exists" "[ -d HelixCode ]"
    run_test "HelixCode config directory exists" "[ -d HelixCode/config ]"
    run_test "HelixCode assets directory exists" "[ -d HelixCode/assets ]"
    
    # Docker compose syntax check
    run_test "docker-compose.helix.yml syntax valid" "docker-compose -f docker-compose.helix.yml config -q"
    
    # Create required directories
    run_test "Create workspace directory" "mkdir -p workspace && [ -d workspace ]"
    run_test "Create projects directory" "mkdir -p projects && [ -d projects ]"
    run_test "Create shared directory" "mkdir -p shared && [ -d shared ]"
    
    # Environment validation
    run_test ".env has required variables" "grep -q 'HELIX_DATABASE_PASSWORD' .env && grep -q 'HELIX_AUTH_JWT_SECRET' .env"
    run_test ".env.example has template variables" "grep -q 'HELIX_API_PORT' .env.example && grep -q 'HELIX_NETWORK_MODE' .env.example"
    
    # Documentation checks
    run_test "DOCKER_SETUP.md is readable" "head -n 5 DOCKER_SETUP.md >/dev/null 2>&1"
    run_test "README_DOCKER.md is readable" "head -n 5 README_DOCKER.md >/dev/null 2>&1"
    
    # Summary
    echo ""
    echo "📊 Quick Test Summary"
    echo "====================="
    echo "Total tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        print_success "🎉 All quick tests passed! Docker setup is ready."
        echo ""
        echo "Next steps:"
        echo "1. Run: ./helix start"
        echo "2. Access services as shown in status output"
        echo "3. Use: ./helix tui for Terminal UI"
        echo "4. Use: ./helix cli for command line"
    else
        print_error "❌ Some tests failed. Check the issues above."
        exit 1
    fi
}

main "$@"