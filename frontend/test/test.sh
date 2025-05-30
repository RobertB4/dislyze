#!/bin/bash

set -e

COMPOSE_FILE="./docker-compose.e2e.yml"
SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR" # Ensure we are in frontend/test

CI_MODE=false # Default CI mode to false

# Parse arguments for --ci flag
if [[ " $@ " =~ " --ci " ]]; then # Check if --ci is among the arguments
  CI_MODE=true
fi

if [ "$CI_MODE" = true ]; then
  echo "CI mode requested. CI environment variable will be set to true."
else
  echo "CI mode not requested. CI environment variable will be set to false (or use host CI if already set and true)."
fi

cleanup() {
  echo "Cleaning up E2E environment..."
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans
  echo "Pruning dangling Docker images..."
  docker image prune -f --filter "dangling=true" || true
}

trap cleanup EXIT SIGINT SIGTERM

echo "Performing initial cleanup..."
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans || true

# Step 1: Start frontend service first to get its IP
echo "Starting frontend service to determine its IP..."
docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate --remove-orphans frontend

# Step 2: Determine Frontend IP dynamically
FRONTEND_CONTAINER_NAME="frontend_e2e" # From docker-compose.e2e.yml container_name
FRONTEND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$FRONTEND_CONTAINER_NAME")

if [ -z "$FRONTEND_IP" ]; then
  echo "Error: Could not determine IP address for frontend container ($FRONTEND_CONTAINER_NAME)."
  exit 1
fi
echo "Frontend container ($FRONTEND_CONTAINER_NAME) IP address: $FRONTEND_IP"

# Step 3: Export the dynamic URL for the backend and Playwright
export DYNAMIC_FRONTEND_URL="http://${FRONTEND_IP}:23000"
echo "Exported DYNAMIC_FRONTEND_URL=${DYNAMIC_FRONTEND_URL}"

# Step 4: Build and start other E2E services.
# The DYNAMIC_FRONTEND_URL will be available to the docker-compose command for the backend.
# We use --no-deps to avoid restarting the frontend if it's already up.
echo "Building and starting other E2E services (backend, postgres, mock-sendgrid, playwright)..."
docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate --remove-orphans --no-deps backend postgres mock-sendgrid playwright

# Health checks
echo "Waiting for services to be healthy..."
MAX_RETRIES=30
RETRY_INTERVAL=5

# Health check for Postgres
echo "Checking Postgres (postgres:5432)..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U testuser_e2e -d testdb_e2e -q; then
    echo "Postgres is ready."
    break
  fi
  echo "Postgres not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "Postgres health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

# Health check for Mock Sendgrid (mock-sendgrid:27000)
echo "Checking Mock Sendgrid (http://mock-sendgrid:27000/)..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T playwright curl --fail --silent --output /dev/null http://mock-sendgrid:27000/; then
    echo "Mock Sendgrid is ready."
    break
  fi
  echo "Mock Sendgrid not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "Mock Sendgrid health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

# Health check for Backend (backend:23001)
echo "Checking Backend (http://backend:23001/health)..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T playwright curl --fail --silent --output /dev/null http://backend:23001/health; then
    echo "Backend is ready."
    break
  fi
  echo "Backend not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "Backend health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

# Health check for Frontend (using the dynamically obtained IP)
echo "Checking Frontend (${DYNAMIC_FRONTEND_URL})..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T playwright curl --fail --silent --output /dev/null "${DYNAMIC_FRONTEND_URL}"; then
    echo "Frontend is ready."
    break
  fi
  echo "Frontend not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "Frontend health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

echo "All services are healthy."

# Run Playwright E2E tests
# The /app directory in the playwright container is the frontend/ directory on the host
# The config file is at /app/test/playwright.e2e.config.ts
# package.json is at /app/package.json
echo "Running Playwright E2E tests targeting ${DYNAMIC_FRONTEND_URL} (CI Mode: $CI_MODE)..."
docker compose -f "$COMPOSE_FILE" exec -T \
  -e CI="${CI_MODE}" \
  -e PLAYWRIGHT_HEADED="${PLAYWRIGHT_HEADED:-false}" \
  -e PLAYWRIGHT_BASE_URL="${DYNAMIC_FRONTEND_URL}" \
  playwright npx playwright test --config test/playwright.e2e.config.js

TEST_EXIT_CODE=$?

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo -e "${GREEN}✅ E2E tests passed successfully.${NC}"
else
  echo -e "${RED}E2E tests failed with exit code $TEST_EXIT_CODE.${NC}"
fi

exit $TEST_EXIT_CODE