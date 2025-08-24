#!/bin/bash

set -e

KEYCLOAK_VERSION="24.0.5"
KEYCLOAK_DIR="keycloak-${KEYCLOAK_VERSION}"
DOWNLOAD_URL="https://github.com/keycloak/keycloak/releases/download/${KEYCLOAK_VERSION}/keycloak-${KEYCLOAK_VERSION}.zip"

echo "Installing Keycloak ${KEYCLOAK_VERSION}..."

# Check if already installed
if [ -d "${KEYCLOAK_DIR}" ]; then
    echo "Keycloak ${KEYCLOAK_VERSION} is already installed in ${KEYCLOAK_DIR}"
    exit 0
fi

# Check if Java is installed
if ! command -v java &> /dev/null; then
    echo "Error: Java is required but not installed. Please install OpenJDK 21 or later."
    echo "On macOS: brew install openjdk@21"
    echo "On Ubuntu: sudo apt install openjdk-21-jdk"
    exit 1
fi

# Check Java version
java_version=$(java -version 2>&1 | head -n1 | cut -d'"' -f2 | cut -d'.' -f1)
if [ "$java_version" -lt 21 ]; then
    echo "Error: Java 21 or later is required. Current version: $java_version"
    exit 1
fi

# Download Keycloak
echo "Downloading Keycloak ${KEYCLOAK_VERSION}..."
if command -v curl &> /dev/null; then
    curl -L -o keycloak-${KEYCLOAK_VERSION}.zip "${DOWNLOAD_URL}"
elif command -v wget &> /dev/null; then
    wget -O keycloak-${KEYCLOAK_VERSION}.zip "${DOWNLOAD_URL}"
else
    echo "Error: Neither curl nor wget is available. Please install one of them."
    exit 1
fi

# Extract Keycloak
echo "Extracting Keycloak..."
if command -v unzip &> /dev/null; then
    unzip -q keycloak-${KEYCLOAK_VERSION}.zip
else
    echo "Error: unzip is required but not installed."
    exit 1
fi

# Create data/import directory
echo "Creating data/import directory..."
mkdir -p "${KEYCLOAK_DIR}/data/import"

# Clean up zip file
rm keycloak-${KEYCLOAK_VERSION}.zip

echo "Keycloak ${KEYCLOAK_VERSION} installed successfully in ${KEYCLOAK_DIR}/"
echo "Run './start.sh' to start Keycloak with the test realm."