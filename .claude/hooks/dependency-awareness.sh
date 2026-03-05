#!/usr/bin/env bash
# PostToolUse hook: dependency awareness
# Shows which files/modules import the edited file, nudges big-picture thinking when cross-module.
#
# Input (stdin): JSON with tool_name and tool_input from Claude Code
# Output (stdout): JSON with result message (or empty for no output)

set -euo pipefail

REPO_ROOT="/Users/robert/Documents/dislyze"

# --- Parse input ---
INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Only run for Edit and Write
if [[ "$TOOL_NAME" != "Edit" && "$TOOL_NAME" != "Write" ]]; then
  exit 0
fi

if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

# --- Skip non-code files ---
case "$FILE_PATH" in
  *.go|*.ts|*.svelte|*.sql) ;;
  *) exit 0 ;;
esac

# Skip test files — they're rarely imported
case "$FILE_PATH" in
  *_test.go|*.test.ts|*.spec.ts) exit 0 ;;
esac

# Skip generated files (SQLC output)
case "$FILE_PATH" in
  */queries/*.go|*/queries/*.sql) exit 0 ;;
esac

# --- Determine what to search for ---
REL_PATH="${FILE_PATH#"$REPO_ROOT/"}"
EDITED_MODULE=$(echo "$REL_PATH" | cut -d'/' -f1)

IMPORTERS=""

find_go_importers() {
  local pkg_dir
  pkg_dir=$(dirname "$REL_PATH")

  local search_dir="$REPO_ROOT/$pkg_dir"
  local mod_name=""
  local mod_dir=""
  while [[ "$search_dir" != "$REPO_ROOT" && "$search_dir" != "/" ]]; do
    if [[ -f "$search_dir/go.mod" ]]; then
      mod_name=$(head -1 "$search_dir/go.mod" | awk '{print $2}')
      mod_dir="${search_dir#"$REPO_ROOT/"}"
      break
    fi
    search_dir=$(dirname "$search_dir")
  done

  if [[ -z "$mod_name" ]]; then
    return
  fi

  local sub_path="${pkg_dir#"$mod_dir"}"
  local import_path="${mod_name}${sub_path}"

  IMPORTERS=$(grep -rn "\"${import_path}\"" --include="*.go" "$REPO_ROOT" 2>/dev/null \
    | grep -v "^${REPO_ROOT}/${pkg_dir}/" \
    | grep -v "/go.mod:" \
    | grep -v "/go.sum:" \
    || true)
}

find_ts_importers() {
  local dir_path
  dir_path=$(dirname "$REL_PATH")
  local filename
  filename=$(basename "$FILE_PATH")
  local filename_no_ext="${filename%.*}"

  local import_path=""

  # Derive canonical import path from file location (mirrors Go import model)
  case "$REL_PATH" in
    lugia-frontend/src/*)
      local sub="${REL_PATH#lugia-frontend/src/}"
      import_path="\$lugia/${sub%.*}"
      ;;
    lugia-frontend/test/*)
      local sub="${REL_PATH#lugia-frontend/test/}"
      import_path="\$lugia-test/${sub%.*}"
      ;;
    giratina-frontend/src/*)
      local sub="${REL_PATH#giratina-frontend/src/}"
      import_path="\$giratina/${sub%.*}"
      ;;
    zoroark/src/lib/utils/*)
      local sub="${REL_PATH#zoroark/src/lib/utils/}"
      import_path="@dislyze/zoroark/${sub%.*}"
      ;;
    zoroark/src/lib/Toast/toast.ts)
      import_path="@dislyze/zoroark/toast"
      ;;
    zoroark/src/lib/Toast/*.svelte)
      local sub="${filename%.svelte}"
      import_path="@dislyze/zoroark/${sub}"
      ;;
    zoroark/src/lib/*.svelte)
      import_path="@dislyze/zoroark/${filename_no_ext}"
      ;;
    *)
      return
      ;;
  esac

  # Strip /index suffix — imports don't include it
  import_path="${import_path%/index}"

  # For .svelte files in frontends, keep the extension in the import path
  # (zoroark deep imports don't use extensions)
  if [[ "$filename" == *.svelte && "$import_path" != @dislyze/* ]]; then
    import_path="${import_path}.svelte"
  fi

  IMPORTERS=$(grep -rn "$import_path" --include="*.ts" --include="*.svelte" "$REPO_ROOT" 2>/dev/null \
    | grep -v "node_modules" \
    | grep -v "\.svelte-kit" \
    | grep -v "/dist/" \
    | grep -v "${FILE_PATH}:" \
    || true)
}

find_sql_importers() {
  if [[ "$REL_PATH" == database/migrations/* ]]; then
    IMPORTERS="STATIC: database migrations are consumed by lugia-backend, giratina-backend, and local dev setup (seed.sql)"
  fi
}

# --- Run the appropriate search ---
case "$FILE_PATH" in
  *.go) find_go_importers ;;
  *.ts|*.svelte) find_ts_importers ;;
  *.sql) find_sql_importers ;;
esac

if [[ -z "$IMPORTERS" ]]; then
  exit 0
fi

# --- Group and format using awk (bash 3.x compatible, no SIGPIPE) ---
if [[ "$IMPORTERS" == STATIC:* ]]; then
  OUTPUT="$IMPORTERS

This file has cross-module consumers. Before continuing, consider: Are you aware of the full blast radius of your changes? Are you fixing the root cause or a symptom? If you are unsure, read .claude/commands/bigpicture.md before continuing."
else
  OUTPUT=$(echo "$IMPORTERS" | awk -v repo_root="$REPO_ROOT/" -v rel_path="$REL_PATH" '
  BEGIN {
    total = 0
  }
  {
    # Extract file path from grep output (path:linenum:content)
    split($0, parts, ":")
    file_path = parts[1]
    line_num = parts[2]

    # Make relative
    sub(repo_root, "", file_path)
    rel_file = file_path

    # Get module (first path component)
    split(rel_file, path_parts, "/")
    mod = path_parts[1]

    total++

    # Track per-module counts
    mod_count[mod]++

    # Store up to 3 samples per module
    if (mod_count[mod] <= 3) {
      if (mod_samples[mod] == "") {
        mod_samples[mod] = rel_file ":" line_num
      } else {
        mod_samples[mod] = mod_samples[mod] ", " rel_file ":" line_num
      }
    }

    # Track unique modules in order
    if (!(mod in seen_mod)) {
      seen_mod[mod] = 1
      mod_order[++mod_idx] = mod
    }
  }
  END {
    printf "You edited: %s\n", rel_path
    printf "This file is imported by %d file(s) across %d service(s):\n", total, mod_idx

    # Sort modules
    for (i = 1; i <= mod_idx; i++) {
      for (j = i + 1; j <= mod_idx; j++) {
        if (mod_order[i] > mod_order[j]) {
          tmp = mod_order[i]
          mod_order[i] = mod_order[j]
          mod_order[j] = tmp
        }
      }
    }

    for (i = 1; i <= mod_idx; i++) {
      m = mod_order[i]
      overflow = ""
      if (mod_count[m] > 3) {
        overflow = " and " (mod_count[m] - 3) " more"
      }
      printf "  %s (%d files): %s%s\n", m, mod_count[m], mod_samples[m], overflow
    }

    if (total >= 3) {
      printf "\nThis file has multiple consumers. Before continuing, consider: Are you aware of the full blast radius of your changes? Are you fixing the root cause or a symptom? If you are unsure, read .claude/commands/bigpicture.md before continuing.\n"
    }
  }
  ')
fi

jq -n --arg message "$OUTPUT" '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $message}}'
