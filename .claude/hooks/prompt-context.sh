#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PRINCIPLES_DIR="$SCRIPT_DIR/../principles"

# Read prompt from stdin JSON
PROMPT=$(cat | jq -r '.prompt // ""')

# Always output core principles
cat "$PRINCIPLES_DIR/core.md"

# Mode detection via keyword triggers — multiple modes can fire
if echo "$PROMPT" | grep -qi "explore"; then
  echo ""
  cat "$PRINCIPLES_DIR/research.md"
fi

if echo "$PROMPT" | grep -qi "plan"; then
  echo ""
  cat "$PRINCIPLES_DIR/planning.md"
fi

if echo "$PROMPT" | grep -qi "implement"; then
  echo ""
  cat "$PRINCIPLES_DIR/implementation.md"
fi

if echo "$PROMPT" | grep -qi "debug"; then
  echo ""
  cat "$PRINCIPLES_DIR/debugging.md"
fi

if echo "$PROMPT" | grep -qi "unit tests\?\|integration tests\?\|e2e tests\?"; then
  echo ""
  cat "$PRINCIPLES_DIR/testing.md"
fi

if echo "$PROMPT" | grep -qi "review"; then
  echo ""
  cat "$PRINCIPLES_DIR/review.md"
fi

exit 0
