FROM node:20-bookworm

# Go toolchain
COPY --from=golang:1.26-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/node/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Adoptium repo (for Java 21, required by Keycloak 24)
RUN curl -fsSL https://packages.adoptium.net/artifactory/api/gpg/key/public \
        | gpg --dearmor -o /etc/apt/trusted.gpg.d/adoptium.gpg \
    && echo "deb https://packages.adoptium.net/artifactory/deb bookworm main" \
        > /etc/apt/sources.list.d/adoptium.list

# System deps + Java 21 (for keycloak mock) + Chromium (for playwright-cli)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git jq ca-certificates curl gnupg make sudo unzip \
    postgresql-client \
    temurin-21-jre \
    chromium \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /opt/google/chrome && ln -sf /usr/bin/chromium /opt/google/chrome/chrome

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
        > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Go tools (lint, migrations, codegen, hot reload, test formatter)
RUN go install github.com/pressly/goose/v3/cmd/goose@latest \
    && go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest \
    && go install github.com/air-verse/air@latest \
    && go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin v2.11.1

# Allow node user to use sudo (required by DinD feature)
RUN echo "node ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/node

# Workspace + fix GOPATH ownership
RUN mkdir -p /workspace && chown node:node /workspace \
    && chown -R node:node /home/node/go

# Playwright CLI (browser automation for agents)
RUN npm install -g @playwright/cli@latest

# Switch to node user
USER node

# Claude Code (native installer)
RUN curl -fsSL https://claude.ai/install.sh | bash
ENV PATH="/home/node/.local/bin:${PATH}"

WORKDIR /workspace
