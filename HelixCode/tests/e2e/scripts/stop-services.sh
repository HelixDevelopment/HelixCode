#!/bin/bash

# E2E Testing Framework - Stop Services Script
# Stops all mock services

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PID_DIR="$E2E_ROOT/.pids"

echo "========================================="
echo "Stopping E2E Mock Services"
echo "========================================="
echo ""

# Stop Mock LLM Provider
if [ -f "$PID_DIR/mock-llm.pid" ]; then
    PID=$(cat "$PID_DIR/mock-llm.pid")
    if ps -p $PID > /dev/null 2>&1; then
        echo "Stopping Mock LLM Provider (PID: $PID)..."
        kill $PID
        sleep 1

        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}Forcing stop...${NC}"
            kill -9 $PID
        fi

        echo -e "${GREEN}✓ Mock LLM Provider stopped${NC}"
    else
        echo -e "${YELLOW}Mock LLM Provider not running${NC}"
    fi
    rm "$PID_DIR/mock-llm.pid"
else
    echo -e "${YELLOW}Mock LLM Provider PID file not found${NC}"
fi

# Stop Mock Slack Service
if [ -f "$PID_DIR/mock-slack.pid" ]; then
    PID=$(cat "$PID_DIR/mock-slack.pid")
    if ps -p $PID > /dev/null 2>&1; then
        echo "Stopping Mock Slack Service (PID: $PID)..."
        kill $PID
        sleep 1

        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}Forcing stop...${NC}"
            kill -9 $PID
        fi

        echo -e "${GREEN}✓ Mock Slack Service stopped${NC}"
    else
        echo -e "${YELLOW}Mock Slack Service not running${NC}"
    fi
    rm "$PID_DIR/mock-slack.pid"
else
    echo -e "${YELLOW}Mock Slack Service PID file not found${NC}"
fi

echo ""
echo -e "${GREEN}All services stopped${NC}"
echo ""
