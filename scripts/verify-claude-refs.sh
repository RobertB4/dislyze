#!/usr/bin/env bash
# Verify that file paths and Makefile targets referenced in CLAUDE.md files actually exist.
# Run from the repo root.

set -euo pipefail

fail=0

# Find all CLAUDE.md files
claude_files=$(find . -name "CLAUDE.md" -not -path "./.git/*" -not -path "./node_modules/*")

for claude_file in $claude_files; do
    dir=$(dirname "$claude_file")

    # Walk up to find the module root (directory containing go.mod or package.json)
    module_root="$dir"
    while [ "$module_root" != "." ] && [ "$module_root" != "/" ]; do
        if [ -f "$module_root/go.mod" ] || [ -f "$module_root/package.json" ]; then
            break
        fi
        module_root=$(dirname "$module_root")
    done

    # Extract backtick-quoted paths, excluding code blocks (``` ... ```)
    paths=$(awk '/^```/{skip=!skip; next} !skip{print}' "$claude_file" \
        | grep -oE '`[^`]+`' \
        | tr -d '`' \
        | grep -E '/' \
        | grep -E '\.(md|sql|json|js|ts|go|yaml|yml|toml|sh|svelte)$|/$' \
        | grep -vE '^(make |http|npm |cd |go |docker |git )' \
        | grep -vE '\*|<|>|\(|\)|=' \
        | sort -u || true)

    for path in $paths; do
        found=0

        # Try multiple resolution strategies
        for base in "$dir" "$module_root" "$dir/src" "$dir/lib" "$module_root/src" "$module_root/lib" "$module_root/src/lib" "."; do
            if [ -e "$base/$path" ]; then
                found=1
                break
            fi
        done

        # For root CLAUDE.md, check if the path exists in any module
        if [ $found -eq 0 ] && [ "$claude_file" = "./CLAUDE.md" ]; then
            for module in lugia-backend giratina-backend jirachi lugia-frontend giratina-frontend zoroark; do
                if [ -e "./$module/$path" ]; then
                    found=1
                    break
                fi
            done
        fi

        if [ $found -eq 0 ]; then
            echo "ERROR: $claude_file references \`$path\` but it does not exist"
            fail=1
        fi
    done

    # Extract `make <target>` references (outside code blocks)
    targets=$(awk '/^```/{skip=!skip; next} !skip{print}' "$claude_file" \
        | grep -oE 'make [a-z][a-z0-9_-]*' \
        | sed 's/^make //' \
        | sort -u || true)

    for target in $targets; do
        # Check Makefiles: walk up from CLAUDE.md dir to root
        found=0
        check_dir="$dir"
        while [ "$check_dir" != "." ] && [ "$check_dir" != "/" ]; do
            if [ -f "$check_dir/Makefile" ] && grep -qE "^${target}:" "$check_dir/Makefile"; then
                found=1
                break
            fi
            check_dir=$(dirname "$check_dir")
        done
        if [ $found -eq 0 ] && [ -f "./Makefile" ] && grep -qE "^${target}:" "./Makefile"; then
            found=1
        fi
        # Also check the module root Makefile
        if [ $found -eq 0 ] && [ -f "$module_root/Makefile" ] && grep -qE "^${target}:" "$module_root/Makefile"; then
            found=1
        fi

        if [ $found -eq 0 ]; then
            echo "ERROR: $claude_file references \`make $target\` but target not found in nearby Makefiles"
            fail=1
        fi
    done
done

if [ $fail -eq 1 ]; then
    exit 1
fi

echo "All CLAUDE.md references validated."
