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
git config user.name "test-scout"
git config user.email "test-scout@dislyze.local"
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

# Read the prompt from the test-scout command
PROMPT_FILE="/workspace/.claude/commands/test-scout.md"
if [ ! -f "$PROMPT_FILE" ]; then
    echo "ERROR: $PROMPT_FILE not found"
    exit 1
fi

PROMPT=$(cat "$PROMPT_FILE")
PROMPT="${PROMPT//\$ARGUMENTS/${AGENT_TARGET:-}}"

echo "=== Starting Claude Code Agent ==="
echo "Model: ${AGENT_MODEL:-opus}"
echo "Target: ${AGENT_TARGET:-<roaming>}"

SCOUT_JSON=$(claude -p "$PROMPT" \
    --dangerously-skip-permissions \
    --model "${AGENT_MODEL:-opus}" \
    --output-format json)

SCOUT_SESSION_ID=$(echo "$SCOUT_JSON" | jq -r '.session_id')
echo "$SCOUT_JSON" | jq -r '.result'

echo "=== Scout Finished (session: ${SCOUT_SESSION_ID}) ==="

# Revert checksum file changes — these are platform-specific noise from running
# Go on linux (container) vs darwin (host), not real dependency changes
git checkout baseline -- go.work.sum 2>/dev/null || true
git checkout baseline -- '*/go.sum' 2>/dev/null || true

# Commit any remaining uncommitted changes
git add -A
if ! git diff --cached --quiet; then
    git commit -q -m "test-scout: uncommitted changes"
fi

# Check if anything changed since baseline (includes commits Claude made)
if [ "$(git rev-parse HEAD)" = "$(git rev-parse baseline)" ]; then
    echo "NO_CHANGES"
    exit 0
fi

# --- Review loop (max 2 iterations) ---

REVIEW_PROMPT_FILE="/workspace/.claude/commands/agent-review.md"
if [ ! -f "$REVIEW_PROMPT_FILE" ]; then
    echo "ERROR: $REVIEW_PROMPT_FILE not found"
    exit 1
fi
REVIEW_PROMPT=$(cat "$REVIEW_PROMPT_FILE")

MAX_REVIEW_ITERATIONS="${AGENT_MAX_REVIEW_ITERATIONS:-2}"
REVIEW_PASSED=false

for i in $(seq 1 "$MAX_REVIEW_ITERATIONS"); do
    echo ""
    echo "=== Review iteration $i/$MAX_REVIEW_ITERATIONS ==="

    REVIEW_OUTPUT=$(claude -p "$REVIEW_PROMPT" \
        --dangerously-skip-permissions \
        --model "${AGENT_MODEL:-opus}"

    echo "$REVIEW_OUTPUT"

    if echo "$REVIEW_OUTPUT" | grep -qi "review passed"; then
        echo ""
        echo "=== Review Passed ==="
        REVIEW_PASSED=true
        break
    fi

    # Feed findings back to scout for fixes (resume scout's session for full context)
    echo ""
    echo "=== Fixing review issues (iteration $i) ==="

    claude -p "The review of your test changes found these issues. Fix them and verify the tests still pass:

${REVIEW_OUTPUT}" \
        --dangerously-skip-permissions \
        --model "${AGENT_MODEL:-opus}" \
        --resume "${SCOUT_SESSION_ID}"

    # Commit fixes
    git add -A
    if ! git diff --cached --quiet; then
        git commit -q -m "test-scout: review fixes (iteration $i)"
    fi
done

# --- Create patches ---

# Commit any final uncommitted changes
git add -A
if ! git diff --cached --quiet; then
    git commit -q -m "test-scout: final changes"
fi

mkdir -p /workspace/patches
git format-patch baseline -o /workspace/patches/ -q

echo "PATCHES_READY"
