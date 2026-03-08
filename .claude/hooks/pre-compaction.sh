#!/usr/bin/env bash
# PreCompact hook: save state before context compaction
# Nudges the agent to persist its current state to its task file
# so it survives the context compression.
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

# --- Select the appropriate prompt ---
if [[ -f "${REPO_ROOT}/${TASK_FILE}" ]]; then
  PROMPT_FILE="$PROMPTS_DIR/pre-compaction.md"
else
  PROMPT_FILE="$PROMPTS_DIR/pre-compaction-lite.md"
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
  exit 0
fi

# --- Read and interpolate the prompt ---
NUDGE_TEXT=$(sed "s|{{TASK_FILE}}|${TASK_FILE}|g" "$PROMPT_FILE")

# PreCompact uses the same output format — additionalContext is added to the conversation
jq -n --arg message "$NUDGE_TEXT" '{
  hookSpecificOutput: {
    hookEventName: "PreCompact",
    additionalContext: $message
  }
}'
