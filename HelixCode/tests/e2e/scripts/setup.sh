#!/bin/bash

# E2E Testing Framework - Quick Start Setup Script
# This script sets up the entire E2E testing environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_ROOT="$(dirname "$SCRIPT_DIR")"

echo "========================================="
echo "HelixCode E2E Testing Framework Setup"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Warning: Docker is not installed (optional for containerized mode)${NC}"
fi

echo -e "${GREEN}✓ Go found: $(go version)${NC}"

# Build orchestrator
echo ""
echo "Building E2E Test Orchestrator..."
cd "$E2E_ROOT/orchestrator"
go mod download
go build -o bin/orchestrator ./cmd/main.go
echo -e "${GREEN}✓ Orchestrator built successfully${NC}"

# Build test bank
echo ""
echo "Building Test Bank..."
cd "$E2E_ROOT/test-bank"
go mod download
echo -e "${GREEN}✓ Test Bank ready${NC}"

# Build mock services
echo ""
echo "Building Mock LLM Provider..."
cd "$E2E_ROOT/mocks/llm-provider"
go mod download
mkdir -p bin
go build -o bin/mock-llm-provider ./cmd/main.go
echo -e "${GREEN}✓ Mock LLM Provider built (12MB)${NC}"

echo ""
echo "Building Mock Slack Service..."
cd "$E2E_ROOT/mocks/slack"
go mod download
mkdir -p bin
go build -o bin/mock-slack ./cmd/main.go
echo -e "${GREEN}✓ Mock Slack Service built (12MB)${NC}"

# Create environment file if it doesn't exist
echo ""
echo "Setting up environment configuration..."
if [ ! -f "$E2E_ROOT/.env" ]; then
    cat > "$E2E_ROOT/.env" << EOF
# E2E Testing Framework Configuration

# Test Orchestrator
E2E_CONCURRENT_TESTS=3
E2E_TIMEOUT=300s
E2E_RETRY_DELAY=1s

# Mock LLM Provider
MOCK_LLM_PORT=8090
MOCK_LLM_DELAY_MS=100
MOCK_LLM_LOGGING=true
MOCK_LLM_FIXTURES=./responses/fixtures.json
MOCK_LLM_DEFAULT_MODEL=mock-gpt-4

# Mock Slack Service
MOCK_SLACK_PORT=8091
MOCK_SLACK_DELAY_MS=50
MOCK_SLACK_LOGGING=true
MOCK_SLACK_STORAGE_CAPACITY=1000

# Test Configuration
TEST_ENV=local
TEST_LOG_LEVEL=info
EOF
    echo -e "${GREEN}✓ Created .env file${NC}"
else
    echo -e "${YELLOW}✓ .env file already exists (skipping)${NC}"
fi

# Make scripts executable
echo ""
echo "Making scripts executable..."
chmod +x "$SCRIPT_DIR"/*.sh
echo -e "${GREEN}✓ Scripts are executable${NC}"

# Summary
echo ""
echo "========================================="
echo -e "${GREEN}Setup Complete!${NC}"
echo "========================================="
echo ""
echo "Available commands:"
echo ""
echo "  ./scripts/start-services.sh    - Start all mock services"
echo "  ./scripts/stop-services.sh     - Stop all services"
echo "  ./scripts/run-tests.sh         - Run E2E tests"
echo "  ./scripts/clean.sh             - Clean up resources"
echo ""
echo "Or use the orchestrator directly:"
echo ""
echo "  cd orchestrator"
echo "  ./bin/orchestrator run --priority critical"
echo "  ./bin/orchestrator list"
echo ""
echo "Documentation:"
echo "  - E2E README: $E2E_ROOT/README.md"
echo "  - Orchestrator: $E2E_ROOT/orchestrator/README.md"
echo "  - Mock LLM: $E2E_ROOT/mocks/llm-provider/README.md"
echo "  - Mock Slack: $E2E_ROOT/mocks/slack/README.md"
echo ""
