#!/bin/bash

set -e

KEYCLOAK_VERSION="24.0.5"
KEYCLOAK_DIR="keycloak-${KEYCLOAK_VERSION}"
PORT=7001

# Check if Keycloak is installed
if [ ! -d "${KEYCLOAK_DIR}" ]; then
    echo "Error: Keycloak not found. Run './install.sh' first."
    exit 1
fi

# Check if realm.json exists
if [ ! -f "realm.json" ]; then
    echo "Error: realm.json not found. Make sure the realm configuration file exists."
    exit 1
fi

echo "Starting Keycloak on port ${PORT}..."

# Copy realm configuration to import directory
echo "Copying realm configuration..."
cp realm.json "${KEYCLOAK_DIR}/data/import/"

# Check if port is already in use
if lsof -i :${PORT} > /dev/null 2>&1; then
    echo "Warning: Port ${PORT} is already in use. Attempting to start anyway..."
fi

# Start Keycloak
cd "${KEYCLOAK_DIR}"

echo "Keycloak will be available at: http://localhost:${PORT}/"
echo "Admin Console: http://localhost:${PORT}/admin/"
echo "Admin credentials: admin / admin123"
echo ""
echo "Press Ctrl+C to stop Keycloak"

KEYCLOAK_ADMIN=admin KEYCLOAK_ADMIN_PASSWORD=admin123 \
./bin/kc.sh start-dev --http-port=${PORT} --import-realm