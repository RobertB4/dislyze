#!/bin/bash
set -euo pipefail

echo "=== Agent Container Starting ==="

# Extract repo from tarball
if [ -f /tmp/repo.tar ]; then
    echo "Extracting repository..."
    tar -xf /tmp/repo.tar -C /workspace
else
    echo "ERROR: /tmp/repo.tar not found. Did the runner mount it?"
    exit 1
fi

cd /workspace

# Initialize git so we can diff and create patches later
git init -q
git config user.name "agent"
git config user.email "agent@dislyze.local"
git add -A
git commit -q -m "baseline"
git tag baseline

# Download Go dependencies
echo "Downloading Go dependencies..."
for mod in jirachi lugia-backend giratina-backend; do
    if [ -d "$mod" ] && [ -f "$mod/go.mod" ]; then
        (cd "$mod" && go mod download) 2>&1 | tail -1 || true
    fi
done

# Install npm dependencies
echo "Installing npm dependencies..."
for dir in zoroark lugia-frontend giratina-frontend; do
    if [ -d "$dir" ] && [ -f "$dir/package.json" ]; then
        (cd "$dir" && npm ci --silent) || true
    fi
done

# Build zoroark (shared component library, needed by frontends)
if [ -d "zoroark" ]; then
    echo "Building zoroark..."
    (cd zoroark && npm run build) || true
fi

PROMPT="${AGENT_PROMPT:?AGENT_PROMPT must be set}"

echo "=== Starting Claude Code Agent ==="
echo "Model: ${AGENT_MODEL:-opus}"
echo ""

TASK_JSON=$(claude -p "$PROMPT" \
    --dangerously-skip-permissions \
    --model "${AGENT_MODEL:-opus}" \
    --output-format json)

TASK_SESSION_ID=$(echo "$TASK_JSON" | jq -r '.session_id')
echo "$TASK_JSON" | jq -r '.result'

echo "=== Task Finished (session: ${TASK_SESSION_ID}) ==="

# Revert checksum file changes — platform-specific noise from running
# Go on linux (container) vs darwin (host)
git checkout baseline -- go.work.sum 2>/dev/null || true
git checkout baseline -- '*/go.sum' 2>/dev/null || true

# Commit any remaining uncommitted changes
git add -A
if ! git diff --cached --quiet; then
    git commit -q -m "agent: uncommitted changes"
fi

# Check if anything changed since baseline
if [ "$(git rev-parse HEAD)" = "$(git rev-parse baseline)" ]; then
    echo "NO_CHANGES"
    exit 0
fi

# --- Review loop (max 3 iterations) ---

REVIEW_PROMPT_FILE="/workspace/.claude/commands/agent-review.md"
if [ ! -f "$REVIEW_PROMPT_FILE" ]; then
    echo "ERROR: $REVIEW_PROMPT_FILE not found"
    exit 1
fi
REVIEW_PROMPT=$(cat "$REVIEW_PROMPT_FILE")

MAX_REVIEW_ITERATIONS="${AGENT_MAX_REVIEW_ITERATIONS:-3}"
REVIEW_PASSED=false

for i in $(seq 1 "$MAX_REVIEW_ITERATIONS"); do
    echo ""
    echo "=== Review iteration $i/$MAX_REVIEW_ITERATIONS ==="

    REVIEW_OUTPUT=$(claude -p "$REVIEW_PROMPT" \
        --dangerously-skip-permissions \
        --model "${AGENT_MODEL:-opus}")

    echo "$REVIEW_OUTPUT"

    if echo "$REVIEW_OUTPUT" | grep -qi "review passed"; then
        echo ""
        echo "=== Review Passed ==="
        REVIEW_PASSED=true
        break
    fi

    # Feed findings back to the task agent for fixes (resume session for full context)
    echo ""
    echo "=== Fixing review issues (iteration $i) ==="

    claude -p "The review of your changes found these issues. Fix them and verify everything still works:

${REVIEW_OUTPUT}" \
        --dangerously-skip-permissions \
        --model "${AGENT_MODEL:-opus}" \
        --resume "${TASK_SESSION_ID}"

    # Commit fixes
    git add -A
    if ! git diff --cached --quiet; then
        git commit -q -m "agent: review fixes (iteration $i)"
    fi
done

# --- Create patches ---

# Commit any final uncommitted changes
git add -A
if ! git diff --cached --quiet; then
    git commit -q -m "agent: final changes"
fi

mkdir -p /workspace/patches
git format-patch baseline -o /workspace/patches/ -q

echo "PATCHES_READY"
