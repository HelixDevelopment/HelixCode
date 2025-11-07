#!/bin/bash

# E2E Testing Framework - Start Services Script
# Starts all mock services required for E2E testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f "$E2E_ROOT/.env" ]; then
    export $(cat "$E2E_ROOT/.env" | grep -v '^#' | xargs)
fi

# Set defaults
MOCK_LLM_PORT=${MOCK_LLM_PORT:-8090}
MOCK_SLACK_PORT=${MOCK_SLACK_PORT:-8091}

PID_DIR="$E2E_ROOT/.pids"
mkdir -p "$PID_DIR"

echo "========================================="
echo "Starting E2E Mock Services"
echo "========================================="
echo ""

# Check if services are already running
if [ -f "$PID_DIR/mock-llm.pid" ]; then
    PID=$(cat "$PID_DIR/mock-llm.pid")
    if ps -p $PID > /dev/null 2>&1; then
        echo -e "${YELLOW}Mock LLM Provider already running (PID: $PID)${NC}"
    else
        rm "$PID_DIR/mock-llm.pid"
    fi
fi

if [ -f "$PID_DIR/mock-slack.pid" ]; then
    PID=$(cat "$PID_DIR/mock-slack.pid")
    if ps -p $PID > /dev/null 2>&1; then
        echo -e "${YELLOW}Mock Slack Service already running (PID: $PID)${NC}"
    else
        rm "$PID_DIR/mock-slack.pid"
    fi
fi

# Start Mock LLM Provider
if [ ! -f "$PID_DIR/mock-llm.pid" ]; then
    echo "Starting Mock LLM Provider on port $MOCK_LLM_PORT..."
    cd "$E2E_ROOT/mocks/llm-provider"
    nohup ./bin/mock-llm-provider > "$PID_DIR/mock-llm.log" 2>&1 &
    echo $! > "$PID_DIR/mock-llm.pid"
    sleep 1

    if ps -p $(cat "$PID_DIR/mock-llm.pid") > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Mock LLM Provider started (PID: $(cat "$PID_DIR/mock-llm.pid"))${NC}"
    else
        echo -e "${RED}✗ Failed to start Mock LLM Provider${NC}"
        cat "$PID_DIR/mock-llm.log"
        exit 1
    fi
fi

# Start Mock Slack Service
if [ ! -f "$PID_DIR/mock-slack.pid" ]; then
    echo "Starting Mock Slack Service on port $MOCK_SLACK_PORT..."
    cd "$E2E_ROOT/mocks/slack"
    nohup ./bin/mock-slack > "$PID_DIR/mock-slack.log" 2>&1 &
    echo $! > "$PID_DIR/mock-slack.pid"
    sleep 1

    if ps -p $(cat "$PID_DIR/mock-slack.pid") > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Mock Slack Service started (PID: $(cat "$PID_DIR/mock-slack.pid"))${NC}"
    else
        echo -e "${RED}✗ Failed to start Mock Slack Service${NC}"
        cat "$PID_DIR/mock-slack.log"
        exit 1
    fi
fi

# Wait for services to be ready
echo ""
echo "Waiting for services to be ready..."
sleep 2

# Health checks
echo ""
echo "Performing health checks..."

if curl -s http://localhost:$MOCK_LLM_PORT/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Mock LLM Provider is healthy${NC}"
else
    echo -e "${RED}✗ Mock LLM Provider health check failed${NC}"
fi

if curl -s http://localhost:$MOCK_SLACK_PORT/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Mock Slack Service is healthy${NC}"
else
    echo -e "${RED}✗ Mock Slack Service health check failed${NC}"
fi

echo ""
echo "========================================="
echo -e "${GREEN}All Services Started Successfully!${NC}"
echo "========================================="
echo ""
echo "Service Endpoints:"
echo "  - Mock LLM Provider: http://localhost:$MOCK_LLM_PORT"
echo "  - Mock Slack Service: http://localhost:$MOCK_SLACK_PORT"
echo ""
echo "Logs:"
echo "  - Mock LLM: $PID_DIR/mock-llm.log"
echo "  - Mock Slack: $PID_DIR/mock-slack.log"
echo ""
echo "To stop services: ./scripts/stop-services.sh"
echo ""
