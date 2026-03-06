#!/usr/bin/env bash
# Verify that backend feature files and frontend page files reference a feature doc.
# Each file must contain "Feature doc: docs/features/<name>.md" in a comment.
# Run from the repo root.

set -euo pipefail

fail=0

# Collect target files:
# - Non-test .go files in backend features/ directories
# - +page.svelte and +page.ts files in frontend routes/ directories
files=$(find \
    lugia-backend/features giratina-backend/features \
    -name '*.go' -not -name '*_test.go' 2>/dev/null || true)

files="$files
$(find \
    lugia-frontend/src/routes giratina-frontend/src/routes \
    \( -name '+page.svelte' -o -name '+page.ts' \) \
    -not -path '*/routes/error/*' 2>/dev/null || true)"

for file in $files; do
    [ -z "$file" ] && continue

    # Check for "Feature doc: docs/features/<name>.md" pattern
    if ! grep -q 'Feature doc: docs/features/.*\.md' "$file"; then
        echo "ERROR: $file is missing a feature doc reference"
        fail=1
        continue
    fi

    # Verify each referenced feature doc exists
    refs=$(grep -oE 'docs/features/[a-z0-9_-]+\.md' "$file" | sort -u)
    for ref in $refs; do
        if [ ! -f "$ref" ]; then
            echo "ERROR: $file references $ref but it does not exist"
            fail=1
        fi
    done
done

if [ $fail -eq 1 ]; then
    exit 1
fi

echo "All feature doc references validated."
