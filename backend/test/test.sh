#!/bin/bash

# Exit on error
set -e

# Print commands
set -x

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "🚀 Starting test environment..."

# Build and start containers
docker compose -f docker-compose.test.yml up -d --build

echo "⏳ Waiting for services to be ready..."

# Wait for PostgreSQL to be ready
sleep 5

echo "🧪 Running tests..."
# Run tests
docker compose -f docker-compose.test.yml exec -T backend go test -v ./test/integration/... -v

# Capture the exit code
TEST_EXIT_CODE=$?

# If tests failed, show backend logs
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}❌ Tests failed!${NC}"
    echo -e "${RED}❌ Tests failed! logs:${NC}"
    docker compose -f docker-compose.test.yml logs backend
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
fi

echo "🧹 Cleaning up..."
# Stop and remove containers
docker compose -f docker-compose.test.yml down

# Exit with the test exit code
exit $TEST_EXIT_CODE 