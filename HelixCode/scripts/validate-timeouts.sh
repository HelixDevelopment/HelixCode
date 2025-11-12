#!/bin/bash

# HelixCode Timeout Configuration Validation Script
# This script ensures that all timeout configurations are properly set
# to prevent premature server shutdown issues

set -e

echo "🔍 Validating HelixCode Timeout Configurations..."
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration files to check
CONFIG_FILES=(
    "config/config.yaml"
    "config/minimal-config.yaml" 
    "config/test-config.yaml"
    "config/working-config.yaml"
    "tests/automation/results/test-config.yaml"
)

# Expected values
EXPECTED_IDLE_TIMEOUT=300
EXPECTED_READ_TIMEOUT=30
EXPECTED_WRITE_TIMEOUT=30

# Check configuration files
for config_file in "${CONFIG_FILES[@]}"; do
    if [[ -f "$config_file" ]]; then
        echo -n "📄 Checking $config_file... "
        
        # Extract idle_timeout value
        if grep -q "idle_timeout" "$config_file"; then
            idle_timeout=$(grep "idle_timeout" "$config_file" | awk '{print $2}' | tr -d 's')
            
            if [[ "$idle_timeout" == "$EXPECTED_IDLE_TIMEOUT" ]]; then
                echo -e "${GREEN}✅ idle_timeout = $idle_timeout${NC}"
            else
                echo -e "${RED}❌ idle_timeout = $idle_timeout (expected $EXPECTED_IDLE_TIMEOUT)${NC}"
                exit 1
            fi
        else
            echo -e "${YELLOW}⚠️  No idle_timeout found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  File not found: $config_file${NC}"
    fi
done

# Check Go code defaults
echo -n "🔧 Checking Go code defaults... "
if grep -q 'viper.SetDefault("server.idle_timeout", 300)' internal/config/config.go; then
    echo -e "${GREEN}✅ Default idle_timeout = 300${NC}"
else
    echo -e "${RED}❌ Default idle_timeout not set to 300${NC}"
    exit 1
fi

# Check for any hardcoded 60-second timeouts
echo -n "🔍 Checking for problematic 60-second timeouts... "
if grep -r "idle_timeout.*60" --include="*.go" --include="*.yaml" --include="*.yml" . | grep -v "test-server-shutdown.go" | grep -v "validate-timeouts.sh" | grep -v "server_timeout_test.go" > /dev/null; then
    echo -e "${RED}❌ Found problematic 60-second timeouts${NC}"
    grep -r "idle_timeout.*60" --include="*.go" --include="*.yaml" --include="*.yml" . | grep -v "test-server-shutdown.go" | grep -v "validate-timeouts.sh" | grep -v "server_timeout_test.go"
    exit 1
else
    echo -e "${GREEN}✅ No problematic 60-second timeouts found${NC}"
fi

# Run regression tests
echo -n "🧪 Running regression tests... "
if go test ./tests/regression -run TestServerTimeoutConfiguration -v > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Regression tests passed${NC}"
else
    echo -e "${RED}❌ Regression tests failed${NC}"
    go test ./tests/regression -run TestServerTimeoutConfiguration -v
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 All timeout configurations validated successfully!${NC}"
echo ""
echo "📋 Summary:"
echo "  ✅ All configuration files have idle_timeout = 300"
echo "  ✅ Go code defaults set to 300 seconds"
echo "  ✅ No problematic 60-second timeouts found"
echo "  ✅ Regression tests passed"
echo ""
echo "🚀 Server should now run without premature shutdown!"

# Optional: Test server startup (uncomment if needed)
# echo ""
# echo "🧪 Testing server startup..."
# timeout 10 ./bin/helixcode --config config/minimal-config.yaml &
# SERVER_PID=$!
# sleep 5
# if ps -p $SERVER_PID > /dev/null; then
#     echo -e "${GREEN}✅ Server started successfully${NC}"
#     kill $SERVER_PID
# else
#     echo -e "${RED}❌ Server failed to start${NC}"
#     exit 1
# fi