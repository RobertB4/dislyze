#!/bin/bash

# Exit on error
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "🚀 Starting test environment..."

# Build and start the containers
docker compose -f docker-compose.test.yml up -d --build

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 5

# Run tests
echo "🧪 Running tests..."
docker compose -f docker-compose.test.yml exec backend go test ./... -v

# Check test result
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
else
    echo -e "${RED}❌ Tests failed!${NC}"
    exit 1
fi

# Clean up
echo "🧹 Cleaning up..."
docker compose -f docker-compose.test.yml down -v

echo "✨ Done!" 