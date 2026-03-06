#!/bin/bash
set -euo pipefail

echo "=== Waiting for Docker daemon ==="
until docker info >/dev/null 2>&1; do
    sleep 1
done

echo "=== Starting PostgreSQL ==="
docker compose -f .devcontainer/services.yml -p dislyze-services up -d --remove-orphans
until pg_isready -h localhost -U postgres -q 2>/dev/null; do
    sleep 1
done

# Install keycloak mock
echo "=== Installing Keycloak mock ==="
if [ -d "keycloak-mock" ] && [ -f "keycloak-mock/install.sh" ]; then
    (cd keycloak-mock && bash install.sh)
fi

echo "=== Downloading Go dependencies ==="
for mod in jirachi lugia-backend giratina-backend; do
    if [ -d "$mod" ] && [ -f "$mod/go.mod" ]; then
        (cd "$mod" && go mod download)
    fi
done

# Fix ownership of volume-mounted node_modules (created as root by Docker)
for dir in zoroark lugia-frontend giratina-frontend sendgrid-mock; do
    if [ -d "$dir/node_modules" ]; then
        sudo chown node:node "$dir/node_modules"
    fi
done

echo "=== Installing npm dependencies ==="
for dir in zoroark lugia-frontend giratina-frontend sendgrid-mock; do
    if [ -d "$dir" ] && [ -f "$dir/package.json" ]; then
        (cd "$dir" && npm ci)
    fi
done

echo "=== Building zoroark ==="
(cd zoroark && npm run build)

echo "=== Setup complete ==="
echo "Run 'make migrate && make seed' to initialize the database"
echo "Run 'make dev' to start all services"
echo "Run 'make verify' to lint + test everything"
echo "Run 'claude' for AI-assisted development"
