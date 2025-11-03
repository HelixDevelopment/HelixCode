#!/bin/bash

# HelixCode Docker Setup Test Script
# This script tests the complete Docker setup for HelixCode

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Function to run a test
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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if port is available
port_available() {
    ! nc -z 127.0.0.1 "$1" 2>/dev/null
}

# Function to wait for service
wait_for_service() {
    local service="$1"
    local port="$2"
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $service to be ready on port $port..."
    
    while [ $attempt -le $max_attempts ]; do
        if nc -z 127.0.0.1 "$port" 2>/dev/null; then
            print_success "$service is ready on port $port"
            return 0
        fi
        
        print_info "Attempt $attempt/$max_attempts: $service not ready yet..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service failed to start within $max_attempts attempts"
    return 1
}

# Main test function
main() {
    echo ""
    echo "🚀 HelixCode Docker Setup Test Suite"
    echo "===================================="
    echo ""
    
    # Test 1: Check Docker installation
    run_test "Docker installation" "command_exists docker"
    
    # Test 2: Check Docker Compose installation
    run_test "Docker Compose installation" "command_exists docker-compose || docker compose version >/dev/null 2>&1"
    
    # Test 3: Check Docker daemon
    run_test "Docker daemon running" "docker info >/dev/null 2>&1"
    
    # Test 4: Check required files
    run_test "Dockerfile exists" "[ -f Dockerfile ]"
    run_test "docker-compose.helix.yml exists" "[ -f docker-compose.helix.yml ]"
    run_test "helix script exists" "[ -f helix ]"
    run_test "docker-entrypoint.sh exists" "[ -f docker-entrypoint.sh ]"
    
    # Test 5: Check script permissions
    run_test "helix script executable" "[ -x helix ]"
    run_test "docker-entrypoint.sh executable" "[ -x docker-entrypoint.sh ]"
    
    # Test 6: Check required directories
    run_test "HelixCode directory exists" "[ -d HelixCode ]"
    run_test "HelixCode config directory exists" "[ -d HelixCode/config ]"
    run_test "HelixCode assets directory exists" "[ -d HelixCode/assets ]"
    
    # Test 7: Check port availability
    run_test "Default API port (8080) available" "port_available 8080"
    run_test "Default SSH port (2222) available" "port_available 2222"
    run_test "Default Web port (3000) available" "port_available 3000"
    
    # Test 8: Check environment file
    if [ -f .env ]; then
        run_test ".env file exists" "true"
        run_test ".env file has required variables" "grep -q 'HELIX_DATABASE_PASSWORD' .env && grep -q 'HELIX_AUTH_JWT_SECRET' .env"
    else
        print_warning ".env file not found - using .env.example for testing"
        cp .env.example .env.test
        run_test ".env.example has required variables" "grep -q 'HELIX_DATABASE_PASSWORD' .env.test && grep -q 'HELIX_AUTH_JWT_SECRET' .env.test"
        rm -f .env.test
    fi
    
    # Test 9: Build Docker images
    print_info "Building Docker images..."
    run_test "Build main HelixCode image" "docker build -t helixcode-test ."
    run_test "Build worker image" "docker build -f Dockerfile.worker -t helixcode-worker-test ."
    
    # Test 10: Test facade script help
    run_test "Facade script help works" "./helix help >/dev/null 2>&1"
    
    # Test 11: Test container startup (if ports available)
    if port_available 8080 && port_available 2222 && port_available 3000; then
        print_info "Testing container startup..."
        
        # Start containers
        run_test "Start containers" "docker-compose -f docker-compose.helix.yml up -d"
        
        # Wait for services to be ready
        sleep 10
        
        # Test 12: Check container status
        run_test "Main container running" "docker ps --format '{{.Names}}' | grep -q '^helixcode$'"
        run_test "PostgreSQL container running" "docker ps --format '{{.Names}}' | grep -q '^helixcode-postgres$'"
        run_test "Redis container running" "docker ps --format '{{.Names}}' | grep -q '^helixcode-redis$'"
        
        # Test 13: Check service connectivity
        run_test "REST API accessible" "curl -f http://localhost:8080/health >/dev/null 2>&1 || curl -f http://localhost:8080/ >/dev/null 2>&1"
        
        # Test 14: Test CLI commands
        run_test "CLI help works" "docker exec helixcode helix help >/dev/null 2>&1"
        run_test "CLI health check" "docker exec helixcode helix cli --health >/dev/null 2>&1"
        
        # Test 15: Test worker connectivity
        run_test "Worker 1 accessible" "docker exec helixcode-worker-1 echo 'test' >/dev/null 2>&1"
        run_test "Worker 2 accessible" "docker exec helixcode-worker-2 echo 'test' >/dev/null 2>&1"
        
        # Test 16: Test shared directory
        run_test "Shared directory created" "[ -f shared/helix-config.json ]"
        run_test "Configuration broadcasted" "grep -q 'server' shared/helix-config.json"
        
        # Stop containers
        print_info "Stopping test containers..."
        docker-compose -f docker-compose.helix.yml down
        
        # Cleanup test images
        docker rmi helixcode-test helixcode-worker-test 2>/dev/null || true
    else
        print_warning "Skipping container startup tests - ports not available"
    fi
    
    # Test 17: Test auto-port functionality
    print_info "Testing auto-port functionality..."
    export HELIX_AUTO_PORT=true
    export HELIX_API_PORT=8080
    export HELIX_SSH_PORT=2222
    export HELIX_WEB_PORT=3000
    
    run_test "Auto-port script function" "./helix status >/dev/null 2>&1"
    
    # Test 18: Test directory creation
    run_test "Workspace directory creation" "mkdir -p workspace && [ -d workspace ]"
    run_test "Projects directory creation" "mkdir -p projects && [ -d projects ]"
    run_test "Shared directory creation" "mkdir -p shared && [ -d shared ]"
    
    # Test 19: Test documentation
    run_test "Documentation exists" "[ -f DOCKER_SETUP.md ]"
    run_test "Documentation is readable" "head -n 10 DOCKER_SETUP.md >/dev/null 2>&1"
    
    # Test 20: Test environment example
    run_test ".env.example exists" "[ -f .env.example ]"
    run_test ".env.example has all variables" "grep -q 'HELIX_API_PORT' .env.example && grep -q 'HELIX_NETWORK_MODE' .env.example"
    
    # Summary
    echo ""
    echo "📊 Test Summary"
    echo "==============="
    echo "Total tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        print_success "🎉 All tests passed! Docker setup is ready for use."
        echo ""
        echo "Next steps:"
        echo "1. Configure your .env file with secure passwords"
        echo "2. Run: ./helix start"
        echo "3. Access services at the displayed URLs"
    else
        print_error "❌ Some tests failed. Please check the issues above."
        exit 1
    fi
}

# Run main function
main "$@"