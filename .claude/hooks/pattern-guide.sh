#!/usr/bin/env bash
# PostToolUse hook: pattern guide
# When a new file is created, points the agent to a canonical example of the same type.
#
# Only fires on Write (new file creation).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# --- Parse input ---
INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')

if [[ "$TOOL_NAME" != "Write" ]]; then
  exit 0
fi

FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

REL_PATH="${FILE_PATH#"$REPO_ROOT/"}"

# --- Match against known patterns ---
TYPE=""
EXAMPLE=""

case "$REL_PATH" in
  lugia-backend/features/*/*_test.go|lugia-backend/features/*/handler.go)
    exit 0
    ;;
  lugia-backend/features/*/*.go)
    TYPE="backend handler"
    EXAMPLE="lugia-backend/features/roles/get_roles.go, lugia-backend/features/roles/create_role.go, lugia-backend/features/roles/update_role.go, lugia-backend/features/roles/delete_role.go"
    ;;
  giratina-backend/features/*/*_test.go|giratina-backend/features/*/handler.go)
    exit 0
    ;;
  giratina-backend/features/*/*.go)
    TYPE="backend handler"
    EXAMPLE="giratina-backend/features/tenants/get_tenants.go, giratina-backend/features/tenants/update_tenant.go"
    ;;
  lugia-frontend/src/routes/**/+page.ts)
    TYPE="page load function"
    EXAMPLE="lugia-frontend/src/routes/settings/roles/+page.ts"
    ;;
  lugia-frontend/src/routes/**/+page.svelte)
    TYPE="page component"
    EXAMPLE="lugia-frontend/src/routes/settings/roles/+page.svelte"
    ;;
  giratina-frontend/src/routes/**/+page.ts)
    TYPE="page load function"
    EXAMPLE="giratina-frontend/src/routes/+page.ts"
    ;;
  giratina-frontend/src/routes/**/+page.svelte)
    TYPE="page component"
    EXAMPLE="giratina-frontend/src/routes/+page.svelte"
    ;;
  zoroark/src/lib/*.svelte)
    TYPE="shared UI component"
    EXAMPLE="zoroark/src/lib/Button.svelte"
    ;;
esac

if [[ -z "$TYPE" ]]; then
  exit 0
fi

# Don't fire if the created file is one of the examples
if echo "$EXAMPLE" | grep -qF "$REL_PATH"; then
  exit 0
fi

MESSAGE="PATTERN GUIDE: You're creating a new ${TYPE} (${REL_PATH}).
Follow the established pattern in: ${EXAMPLE}
Read this file before continuing if you haven't already."

jq -n --arg message "$MESSAGE" '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $message}}'
