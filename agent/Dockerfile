FROM node:20-bookworm

# Copy Go toolchain from official image
COPY --from=golang:1.24-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/node/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# System deps + Docker CLI
RUN apt-get update && apt-get install -y --no-install-recommends \
    git jq ca-certificates curl gnupg make \
    && install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && chmod a+r /etc/apt/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" > /etc/apt/sources.list.d/docker.list \
    && apt-get update && apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Workspace + entrypoint (while still root)
RUN mkdir -p /workspace && chown node:node /workspace
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to node user for Claude Code install
USER node

# Claude Code CLI (native installer — installs to ~/.local/bin/)
RUN curl -fsSL https://claude.ai/install.sh | bash
ENV PATH="/home/node/.local/bin:${PATH}"

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
