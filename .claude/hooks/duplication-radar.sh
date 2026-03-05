#!/usr/bin/env bash
# PostToolUse hook: duplication radar
# When a new function is added, searches the codebase for similarly named functions.
#
# Detects: function name( (TS/JS) and func Name( (Go)
# Compares old_string vs new_string to find newly added functions.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# --- Parse input ---
INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

# --- Skip non-code files ---
case "$FILE_PATH" in
  *.go|*.ts|*.svelte) ;;
  *) exit 0 ;;
esac

# Skip test files
case "$FILE_PATH" in
  *_test.go|*.test.ts|*.spec.ts) exit 0 ;;
esac

# Skip generated files
case "$FILE_PATH" in
  */queries/*.go|*/queries/*.sql) exit 0 ;;
esac

# --- Extract old and new function names ---
extract_func_names() {
  local text="$1"
  local lang="$2"

  if [[ "$lang" == "go" ]]; then
    # Go: func FunctionName(  or  func (receiver) FunctionName(
    echo "$text" | grep -oE 'func[[:space:]]+(\([^)]*\)[[:space:]]+)?[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(' \
      | sed -E 's/func[[:space:]]+(\([^)]*\)[[:space:]]+)?([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\(/\2/' \
      || true
  else
    # TS/Svelte: function functionName(  or  async function functionName(
    echo "$text" | grep -oE '(async[[:space:]]+)?function[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(' \
      | sed -E 's/(async[[:space:]]+)?function[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\(/\2/' \
      || true
  fi
}

# Determine language
LANG_TYPE="ts"
case "$FILE_PATH" in
  *.go) LANG_TYPE="go" ;;
esac

if [[ "$TOOL_NAME" == "Edit" ]]; then
  OLD_STRING=$(echo "$INPUT" | jq -r '.tool_input.old_string // empty')
  NEW_STRING=$(echo "$INPUT" | jq -r '.tool_input.new_string // empty')

  OLD_FUNCS=$(extract_func_names "$OLD_STRING" "$LANG_TYPE")
  NEW_FUNCS=$(extract_func_names "$NEW_STRING" "$LANG_TYPE")

  # Find functions in new but not in old
  if [[ -z "$NEW_FUNCS" ]]; then
    exit 0
  fi

  ADDED_FUNCS=""
  while IFS= read -r func_name; do
    [[ -z "$func_name" ]] && continue
    if ! echo "$OLD_FUNCS" | grep -qxF "$func_name"; then
      ADDED_FUNCS="${ADDED_FUNCS}${func_name}"$'\n'
    fi
  done <<< "$NEW_FUNCS"

elif [[ "$TOOL_NAME" == "Write" ]]; then
  CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // empty')
  ADDED_FUNCS=$(extract_func_names "$CONTENT" "$LANG_TYPE")
else
  exit 0
fi

# Trim trailing newline
ADDED_FUNCS=$(echo "$ADDED_FUNCS" | sed '/^$/d')

if [[ -z "$ADDED_FUNCS" ]]; then
  exit 0
fi

# --- Extract noun stem from a function name ---
# Strips common verb prefixes: handleLogout -> Logout, formatDate -> Date, getUser -> User
extract_stem() {
  local name="$1"
  # Strip leading lowercase verb prefixes (camelCase boundary = first uppercase after prefix)
  local stem
  stem=$(echo "$name" | sed -E 's/^([Hh]andle|[Gg]et|[Ss]et|[Ff]etch|[Ff]ormat|[Pp]arse|[Nn]ormalize|[Vv]alidate|[Cc]reate|[Uu]pdate|[Dd]elete|[Rr]emove|[Cc]heck|[Ii]s|[Hh]as|[Cc]an|[Ss]hould|[Ff]ind|[Ll]oad|[Ss]ave|[Ss]end|[Ss]how|[Hh]ide|[Tt]oggle|[Ee]nable|[Dd]isable|[Ii]nit|[Ss]etup|[Mm]ake|[Bb]uild|[Rr]ender|[Pp]rocess|[Cc]onvert|[Ee]xtract|[Rr]esolve|[Ee]nsure|[Vv]erify|[Cc]ompute|[Cc]alculate|[Tt]ransform|[Aa]pply|[Rr]eset|[Cc]lear)//')
  # Only use stem if it's meaningful (at least 4 chars and different from original)
  if [[ "${#stem}" -ge 4 && "$stem" != "$name" ]]; then
    echo "$stem"
  fi
}

# --- Search for similar functions in the codebase ---
REL_PATH="${FILE_PATH#"$REPO_ROOT/"}"
OUTPUT=""

while IFS= read -r func_name; do
  [[ -z "$func_name" ]] && continue

  # 1. Exact match: search for this exact function name declared elsewhere
  EXACT_MATCHES=$(grep -rn -E "(function|func)[[:space:]]+(\([^)]*\)[[:space:]]+)?${func_name}[[:space:]]*\(" \
    --include="*.go" --include="*.ts" --include="*.svelte" \
    "$REPO_ROOT" 2>/dev/null \
    | grep -v "node_modules" \
    | grep -v "\.svelte-kit" \
    | grep -v "/dist/" \
    | grep -v "${FILE_PATH}:" \
    || true)

  if [[ -n "$EXACT_MATCHES" ]]; then
    MATCH_COUNT=$(echo "$EXACT_MATCHES" | wc -l | tr -d ' ')
    SAMPLE=$(echo "$EXACT_MATCHES" | head -5 | while IFS= read -r line; do
      rel_line="${line#"$REPO_ROOT/"}"
      echo "    $rel_line"
    done)

    OVERFLOW=""
    if [[ "$MATCH_COUNT" -gt 5 ]]; then
      OVERFLOW=" (and $((MATCH_COUNT - 5)) more)"
    fi

    OUTPUT="${OUTPUT}Exact match: '${func_name}' already exists (${MATCH_COUNT} match(es))${OVERFLOW}:
${SAMPLE}
"
    continue
  fi

  # 2. Stem match: strip verb prefix and search for functions with the same noun stem
  STEM=$(extract_stem "$func_name")
  if [[ -z "$STEM" ]]; then
    continue
  fi

  # For stem matches, only exclude the new function's own declaration (not the whole file)
  STEM_MATCHES=$(grep -rn -E "(function|func)[[:space:]]+(\([^)]*\)[[:space:]]+)?[A-Za-z_]*${STEM}[[:space:]]*\(" \
    --include="*.go" --include="*.ts" --include="*.svelte" \
    "$REPO_ROOT" 2>/dev/null \
    | grep -v "node_modules" \
    | grep -v "\.svelte-kit" \
    | grep -v "/dist/" \
    | grep -v "${func_name}" \
    || true)

  if [[ -n "$STEM_MATCHES" ]]; then
    MATCH_COUNT=$(echo "$STEM_MATCHES" | wc -l | tr -d ' ')
    SAMPLE=$(echo "$STEM_MATCHES" | head -5 | while IFS= read -r line; do
      rel_line="${line#"$REPO_ROOT/"}"
      echo "    $rel_line"
    done)

    OVERFLOW=""
    if [[ "$MATCH_COUNT" -gt 5 ]]; then
      OVERFLOW=" (and $((MATCH_COUNT - 5)) more)"
    fi

    OUTPUT="${OUTPUT}Similar to '${func_name}' (stem '${STEM}') — ${MATCH_COUNT} function(s) with same noun${OVERFLOW}:
${SAMPLE}
"
  fi
done <<< "$ADDED_FUNCS"

if [[ -z "$OUTPUT" ]]; then
  exit 0
fi

HEADER="DUPLICATION RADAR: You added new function(s) in ${REL_PATH}, a similar function may already exist:

${OUTPUT}
Before continuing, consider: Are you introducing tech debt by creating a duplication of an existing function? If you are unsure, read .claude/commands/bigpicture.md before continuing."

jq -n --arg message "$HEADER" '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $message}}'
