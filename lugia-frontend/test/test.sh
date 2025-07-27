#!/bin/bash

set -e

COMPOSE_FILE="./docker-compose.e2e.yml"
SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR" # Ensure we are in lugia-frontend/test

CI_MODE=false
UI_MODE=false


for arg in "$@"
do
    case $arg in
        --ci)
        CI_MODE=true
        shift
        ;;
        --ui)
        UI_MODE=true
        shift
        ;;
    esac
done

if [ "$CI_MODE" = true ]; then
  echo "CI mode requested. CI environment variable will be set to true."
else
  echo "CI mode not requested. CI environment variable will be set to false (or use host CI if already set and true)."
fi

if [ "$UI_MODE" = true ]; then
  echo "Playwright UI mode requested. Attempting to listen on all interfaces."
fi

cleanup() {
  echo "Cleaning up E2E environment..."
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans
  echo "Pruning dangling Docker images..."
  docker image prune -f --filter "dangling=true" || true
}

trap cleanup EXIT SIGINT SIGTERM

echo "Performing initial cleanup..."
echo "Docker info before cleanup:"
docker system df
echo "Current Docker images:"
docker images | head -10
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans || true

# Step 1: Start lugia-frontend service first to get its IP
echo "Starting lugia-frontend service to determine its IP..."
echo "Building lugia-frontend with verbose output..."
docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate --remove-orphans lugia-frontend --verbose

# Step 2: Determine lugia-frontend IP dynamically
FRONTEND_CONTAINER_NAME="lugia-frontend-e2e"
FRONTEND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$FRONTEND_CONTAINER_NAME")

if [ -z "$FRONTEND_IP" ]; then
  echo "Error: Could not determine IP address for lugia-frontend container ($FRONTEND_CONTAINER_NAME)."
  exit 1
fi
echo "lugia-frontend container ($FRONTEND_CONTAINER_NAME) IP address: $FRONTEND_IP"

# Step 3: Export the dynamic URL for the backend and Playwright
export DYNAMIC_FRONTEND_URL="http://${FRONTEND_IP}:23000"
echo "Exported DYNAMIC_FRONTEND_URL=${DYNAMIC_FRONTEND_URL}"

# Step 4: Build and start other E2E services.
# The DYNAMIC_FRONTEND_URL will be available to the docker-compose command for the backend.
# We use --no-deps to avoid restarting the lugia-frontend if it's already up.
echo "Building and starting other E2E services (lugia-backend, postgres, mock-sendgrid, playwright)..."
echo "Available Docker images before build:"
docker images | grep -E "(lugia|test-lugia)" || echo "No lugia images found"
echo "Running docker compose up with verbose output..."
docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate --remove-orphans --no-deps lugia-backend postgres mock-sendgrid playwright --verbose

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

# Health check for Backend (lugia-backend:23001)
echo "Checking lugia-backend (http://lugia-backend:23001/health)..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T playwright curl --fail --silent --output /dev/null http://lugia-backend:23001/health; then
    echo "lugia-backend is ready."
    break
  fi
  echo "lugia-backend not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "lugia-backend health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

# Health check for lugia-frontend (using the dynamically obtained IP)
echo "Checking lugia-frontend (${DYNAMIC_FRONTEND_URL})..."
for i in $(seq 1 $MAX_RETRIES); do
  if docker compose -f "$COMPOSE_FILE" exec -T playwright curl --fail --silent --output /dev/null "${DYNAMIC_FRONTEND_URL}"; then
    echo "lugia-frontend is ready."
    break
  fi
  echo "lugia-frontend not ready, retrying in $RETRY_INTERVAL seconds... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "lugia-frontend health check failed after $MAX_RETRIES retries."
    exit 1
  fi
done

echo "All services are healthy."

echo "Changing ownership of /app in Playwright container to pwuser..."
docker compose -f "$COMPOSE_FILE" exec -T -u root playwright chown -R pwuser:pwuser /app
CHOWN_EXIT_CODE=$?
if [ $CHOWN_EXIT_CODE -ne 0 ]; then
  echo "Error: chown in Playwright container failed with exit code $CHOWN_EXIT_CODE."
  exit $CHOWN_EXIT_CODE
fi
echo "/app ownership changed successfully."

echo "Installing project Playwright dependencies in Playwright container..."
# Make npm ci verbose and capture its exit code
docker compose -f "$COMPOSE_FILE" exec -T playwright npm ci --no-audit --no-fund
NPM_CI_EXIT_CODE=$?

if [ $NPM_CI_EXIT_CODE -ne 0 ]; then
  echo "Error: npm ci in Playwright container failed with exit code $NPM_CI_EXIT_CODE."
  exit $NPM_CI_EXIT_CODE
fi
echo "npm ci completed successfully in Playwright container."

# Run Playwright E2E tests
echo "Running Playwright E2E tests targeting ${DYNAMIC_FRONTEND_URL} (CI Mode: $CI_MODE, UI Mode: $UI_MODE)..."

PLAYWRIGHT_COMMAND_BASE="npx playwright test"
PLAYWRIGHT_COMMAND_ARGS="--config test/playwright.e2e.config.js"

DOCKER_EXEC_ENV_VARS=("-e" "CI=${CI_MODE}")
DOCKER_EXEC_ENV_VARS+=("-e" "PLAYWRIGHT_HEADED=false")
DOCKER_EXEC_ENV_VARS+=("-e" "PLAYWRIGHT_BASE_URL=${DYNAMIC_FRONTEND_URL}")

COMMAND_PREFIX=""

if [ "$UI_MODE" = true ]; then
  PLAYWRIGHT_COMMAND_ARGS="$PLAYWRIGHT_COMMAND_ARGS --ui --ui-host 0.0.0.0 --ui-port 8080"
  # Ensure xvfb-run prefix is active for UI mode
  COMMAND_PREFIX="xvfb-run --auto-servernum --server-args='-screen 0 1280x1024x24' "
fi

COMMAND_TO_RUN="${COMMAND_PREFIX}${PLAYWRIGHT_COMMAND_BASE} ${PLAYWRIGHT_COMMAND_ARGS}"

docker compose -f "$COMPOSE_FILE" exec -T \
  $(printf " %s" "${DOCKER_EXEC_ENV_VARS[@]}") \
  playwright sh -c "$COMMAND_TO_RUN"

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
