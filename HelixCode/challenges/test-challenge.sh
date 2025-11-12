#!/bin/bash

# HelixCode Multi-Agent API Challenge Test Script
# This script tests the basic setup and functionality of the challenge

set -e

echo "🧪 Testing HelixCode Multi-Agent API Challenge Setup"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
HELIX_URL="http://localhost:8080"
CHALLENGE_DIR="/Volumes/T7/Projects/HelixCode/HelixCode/challenges"

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        exit 1
    fi
}

# Function to check if server is running
check_server() {
    echo -e "${YELLOW}🔍 Checking if HelixCode server is running...${NC}"
    
    # Try to connect to the health endpoint
    if command -v curl >/dev/null 2>&1; then
        response=$(curl -s -o /dev/null -w "%{http_code}" "$HELIX_URL/health" 2>/dev/null || echo "000")
        if [ "$response" = "200" ]; then
            print_status 0 "Server is running and healthy"
        else
            print_status 1 "Server is not responding (HTTP $response)"
        fi
    else
        echo -e "${YELLOW}⚠️  curl not available, skipping server check${NC}"
    fi
}

# Function to check challenge files
check_files() {
    echo -e "${YELLOW}📁 Checking challenge files...${NC}"
    
    required_files=(
        "multi-agent-api-challenge.md"
        "multi-agent-api-challenge-solution.go"
        "README.md"
    )
    
    for file in "${required_files[@]}"; do
        if [ -f "$CHALLENGE_DIR/$file" ]; then
            print_status 0 "Found $file"
        else
            print_status 1 "Missing required file: $file"
        fi
    done
}

# Function to check Go environment
check_go() {
    echo -e "${YELLOW}🔧 Checking Go environment...${NC}"
    
    if command -v go >/dev/null 2>&1; then
        go_version=$(go version | awk '{print $3}')
        print_status 0 "Go installed: $go_version"
    else
        print_status 1 "Go is not installed or not in PATH"
    fi
}

# Function to compile challenge solution
compile_solution() {
    echo -e "${YELLOW}🔨 Compiling challenge solution...${NC}"
    
    cd "$CHALLENGE_DIR"
    
    if go build -o challenge-solution multi-agent-api-challenge-solution.go 2>/dev/null; then
        print_status 0 "Challenge solution compiled successfully"
        # Clean up
        rm -f challenge-solution
    else
        print_status 1 "Failed to compile challenge solution"
    fi
}

# Function to validate challenge structure
validate_structure() {
    echo -e "${YELLOW}📋 Validating challenge structure...${NC}"
    
    # Check that the solution implements the required interfaces
    if grep -q "type Agent interface" "$CHALLENGE_DIR/multi-agent-api-challenge-solution.go"; then
        print_status 0 "Agent interface defined"
    else
        print_status 1 "Agent interface not found in solution"
    fi
    
    if grep -q "type TaskManager struct" "$CHALLENGE_DIR/multi-agent-api-challenge-solution.go"; then
        print_status 0 "TaskManager struct defined"
    else
        print_status 1 "TaskManager struct not found in solution"
    fi
    
    if grep -q "func.*authenticate" "$CHALLENGE_DIR/multi-agent-api-challenge-solution.go"; then
        print_status 0 "Authentication method implemented"
    else
        print_status 1 "Authentication method not found"
    fi
}

# Function to display next steps
show_next_steps() {
    echo ""
    echo -e "${GREEN}🎉 All checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start the HelixCode server:"
    echo "   cd /Volumes/T7/Projects/HelixCode/HelixCode"
    echo "   export HELIX_DATABASE_PASSWORD=helixcode123"
    echo "   ./bin/helixcode"
    echo ""
    echo "2. In a new terminal, run the challenge:"
    echo "   cd /Volumes/T7/Projects/HelixCode/HelixCode/challenges"
    echo "   go run multi-agent-api-challenge-solution.go"
    echo ""
    echo "3. Review the challenge specification:"
    echo "   cat multi-agent-api-challenge.md"
    echo ""
}

# Main test execution
main() {
    echo ""
    check_server
    echo ""
    check_files
    echo ""
    check_go
    echo ""
    compile_solution
    echo ""
    validate_structure
    echo ""
    show_next_steps
}

# Run the tests
main