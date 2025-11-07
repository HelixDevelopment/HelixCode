#!/bin/bash

# E2E Testing Framework - Run Tests Script
# Executes E2E tests using the orchestrator

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

echo "========================================="
echo "Running E2E Tests"
echo "========================================="
echo ""

# Check if services are running
PID_DIR="$E2E_ROOT/.pids"
MOCK_LLM_PORT=${MOCK_LLM_PORT:-8090}
MOCK_SLACK_PORT=${MOCK_SLACK_PORT:-8091}

SERVICES_STARTED=false

if [ ! -f "$PID_DIR/mock-llm.pid" ] || [ ! -f "$PID_DIR/mock-slack.pid" ]; then
    echo -e "${YELLOW}Mock services not running. Starting them...${NC}"
    "$SCRIPT_DIR/start-services.sh"
    SERVICES_STARTED=true
    echo ""
fi

# Parse arguments
PRIORITY=""
TAGS=""
PARALLEL=""
OUTPUT_DIR="$E2E_ROOT/test-results"

while [[ $# -gt 0 ]]; do
    case $1 in
        --priority)
            PRIORITY="$2"
            shift 2
            ;;
        --tags)
            TAGS="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL="$2"
            shift 2
            ;;
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build command
CMD="$E2E_ROOT/orchestrator/bin/orchestrator run"

if [ -n "$PRIORITY" ]; then
    CMD="$CMD --priority $PRIORITY"
fi

if [ -n "$TAGS" ]; then
    CMD="$CMD --tags $TAGS"
fi

if [ -n "$PARALLEL" ]; then
    CMD="$CMD --parallel $PARALLEL"
fi

CMD="$CMD --output $OUTPUT_DIR"

# Run tests
echo "Executing tests..."
echo "Command: $CMD"
echo ""

cd "$E2E_ROOT/orchestrator"
$CMD

EXIT_CODE=$?

# Display results
echo ""
echo "========================================="
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}Tests Completed Successfully!${NC}"
else
    echo -e "${RED}Tests Failed (Exit Code: $EXIT_CODE)${NC}"
fi
echo "========================================="
echo ""
echo "Results available at: $OUTPUT_DIR"
echo ""

# Offer to stop services if we started them
if [ "$SERVICES_STARTED" = true ]; then
    echo "Services were started by this script."
    echo "To stop them, run: ./scripts/stop-services.sh"
    echo ""
fi

exit $EXIT_CODE
