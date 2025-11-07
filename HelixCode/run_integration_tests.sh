#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         HELIXCODE INTEGRATION TESTS RUNNER                   ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Start Docker services
echo -e "${YELLOW}🐳 Starting Docker test services...${NC}"
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Check PostgreSQL health
echo -e "${YELLOW}🔍 Checking PostgreSQL health...${NC}"
until docker exec helixcode-postgres-test pg_isready -U helix_test -d helix_test > /dev/null 2>&1; do
  echo -e "${YELLOW}   Waiting for PostgreSQL...${NC}"
  sleep 2
done
echo -e "${GREEN}✅ PostgreSQL is ready${NC}"

# Check Redis health
echo -e "${YELLOW}🔍 Checking Redis health...${NC}"
until docker exec helixcode-redis-test redis-cli --raw incr ping > /dev/null 2>&1; do
  echo -e "${YELLOW}   Waiting for Redis...${NC}"
  sleep 2
done
echo -e "${GREEN}✅ Redis is ready${NC}"

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         RUNNING INTEGRATION TESTS                            ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Run integration tests
go test -v -tags=integration ./internal/database -count=1

TEST_EXIT_CODE=$?

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         TEST RESULTS                                         ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All integration tests PASSED${NC}"
else
    echo -e "${RED}❌ Some integration tests FAILED${NC}"
fi

# Ask to stop Docker services
echo ""
read -p "Do you want to stop Docker test services? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}🛑 Stopping Docker test services...${NC}"
    docker-compose -f docker-compose.test.yml down
    echo -e "${GREEN}✅ Services stopped${NC}"
else
    echo -e "${YELLOW}ℹ️  Services left running. Stop them with: docker-compose -f docker-compose.test.yml down${NC}"
fi

exit $TEST_EXIT_CODE
