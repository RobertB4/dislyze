#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

cleanup() {
    echo "🧹 Cleaning up..."
    # Stop and remove containers
    docker compose -p lugia-backend-integration -f docker-compose.integration.yml down --volumes
}

# Set trap to run cleanup on exit (success, failure, or interruption)
trap cleanup EXIT

echo "🚀 Starting test environment..."

# Build and start containers
docker compose -p lugia-backend-integration -f docker-compose.integration.yml up -d --build

echo "⏳ Waiting for services to be ready..."

# Wait for PostgreSQL to be ready
sleep 5

echo "🧪 Running tests..."
# Run tests
docker compose -p lugia-backend-integration -f docker-compose.integration.yml exec lugia-backend sh -c "go test ./test/integration/... -json -v -p 1 -parallel 1 2>&1 | gotestfmt -hide=empty-packages,successful-tests"

# Capture the exit code
TEST_EXIT_CODE=$?

# If tests failed, show lugia-backend logs
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}❌ Tests failed!${NC}"
    echo -e "${RED}❌ Tests failed! logs:${NC}"
    docker compose -p lugia-backend-integration -f docker-compose.integration.yml logs lugia-backend
else
    echo -e "${GREEN}✅ Integration tests passed successfully.${NC}"
fi

# Exit with the test exit code
exit $TEST_EXIT_CODE 