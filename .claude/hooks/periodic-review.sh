#!/usr/bin/env bash
# PostToolUse hook: periodic review guardrail
# Every 7 minutes, nudges the agent to pause and run `make periodic-review`.
# Uses a state file keyed by session_id to track timing.
#
# Input (stdin): JSON with session_id, tool_name, etc.
# Output (stdout): JSON with additionalContext when nudge fires

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/../prompts"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

NUDGE_INTERVAL_SECONDS=420  # 7 minutes

# --- Parse input ---
INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // empty')

if [[ -z "$SESSION_ID" ]]; then
  exit 0
fi

STATE_FILE="/tmp/claude-guardrails-${SESSION_ID}.json"
TASK_FILE="docs/tasks/TASK-${SESSION_ID}.md"
HAS_TASK_FILE=false

if [[ -f "${REPO_ROOT}/${TASK_FILE}" ]]; then
  HAS_TASK_FILE=true
fi

# --- Initialize state file if it doesn't exist ---
if [[ ! -f "$STATE_FILE" ]]; then
  NOW=$(date +%s)
  jq -n --arg ts "$NOW" '{"last_nudge_ts": $ts}' > "$STATE_FILE"
  exit 0
fi

# --- Check if enough time has passed ---
NOW=$(date +%s)
LAST_NUDGE=$(jq -r '.last_nudge_ts // "0"' "$STATE_FILE")
ELAPSED=$(( NOW - LAST_NUDGE ))

if [[ "$ELAPSED" -lt "$NUDGE_INTERVAL_SECONDS" ]]; then
  exit 0
fi

# --- Time to nudge: update timestamp ---
jq --arg ts "$NOW" '.last_nudge_ts = $ts' "$STATE_FILE" > "${STATE_FILE}.tmp" \
  && mv "${STATE_FILE}.tmp" "$STATE_FILE"

# --- Select the appropriate prompt ---
if [[ "$HAS_TASK_FILE" == "true" ]]; then
  PROMPT_FILE="$PROMPTS_DIR/periodic-review.md"
else
  PROMPT_FILE="$PROMPTS_DIR/periodic-review-lite.md"
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
  exit 0
fi

NUDGE_TEXT=$(cat "$PROMPT_FILE")

if [[ "$HAS_TASK_FILE" == "true" ]]; then
  NUDGE_TEXT="${NUDGE_TEXT} Task file: ${TASK_FILE}"
fi

jq -n --arg message "$NUDGE_TEXT" '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: $message
  }
}'
