#!/usr/bin/env bash
# SessionStart hook (matcher: compact): re-orient after context compaction
# Fires when a session resumes after compaction, nudging the agent
# to re-read its task file and re-orient.
#
# Input (stdin): JSON with session_id, etc.
# Output (stdout): context message for the agent

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/../prompts"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Parse input ---
INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // empty')

if [[ -z "$SESSION_ID" ]]; then
  exit 0
fi

TASK_FILE="docs/tasks/TASK-${SESSION_ID}.md"

# --- Reset the periodic review timer after compaction ---
# The agent is about to re-orient, so don't nudge again immediately
NOW=$(date +%s)
STATE_FILE="/tmp/claude-guardrails-${SESSION_ID}.json"
jq -n --arg ts "$NOW" '{"last_nudge_ts": $ts}' > "$STATE_FILE"

# --- Select the appropriate prompt ---
if [[ -f "${REPO_ROOT}/${TASK_FILE}" ]]; then
  PROMPT_FILE="$PROMPTS_DIR/post-compaction.md"
else
  PROMPT_FILE="$PROMPTS_DIR/post-compaction-lite.md"
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
  exit 0
fi

# --- Read and interpolate the prompt ---
NUDGE_TEXT=$(sed "s|{{TASK_FILE}}|${TASK_FILE}|g" "$PROMPT_FILE")

# SessionStart stdout is added as context that Claude can see and act on
echo "$NUDGE_TEXT"
