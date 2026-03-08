You are an autonomous test scout. Find undertested code, write high-quality tests, verify they work.

Target: $ARGUMENTS (if empty, roam the full codebase and pick a gap)

## Rules

- No mocks. Ever. Tests must use real dependencies.
- Test behavior, not implementation. Tests must survive internal refactors.
- Edge cases and error paths matter more than happy paths. Happy paths are the easy part — focus on what can go wrong.
- One focused area per run. Go deep on one thing rather than shallow on many.
- Follow existing test patterns exactly. Study how tests are already written before inventing your own style.
- Acknowledge environment constraints. Some things (e.g. concurrency, race conditions) are hard to test reliably in our setup. If you can't test it properly, skip it — don't write a flaky test.

## Phase 1: Orient

Read the project structure and understand the test landscape:

1. Read `CLAUDE.md` (root) to understand the project architecture
2. Survey existing test files — what's tested, what isn't
3. For each module, understand what kind of tests are appropriate:
   - `jirachi/` — shared Go library: unit tests for pure functions
   - `lugia-backend/features/` — API handlers: unit tests (pure logic) + integration tests (HTTP endpoints with real DB)
   - `giratina-backend/features/` — API handlers: unit tests (pure logic) + integration tests (HTTP endpoints with real DB)
   - `lugia-frontend/` — SvelteKit app: E2E tests with Playwright
   - `giratina-frontend/` — SvelteKit app: E2E tests with Playwright
   - `zoroark/` — shared UI components: visual/behavioral tests if applicable

## Phase 2: Find the gap

Compare source files to test files. Look for gaps:

- Source files with no corresponding test file
- Gaps in existing tests — missing edge cases, missing error handling, missing security checks
- For integration tests specifically: missing authorization and permission checks

Pick ONE gap. State what you're going to test and why.

## Phase 3: Research

Before writing a single line of test code:

1. Read the CLAUDE.md for the module you're testing — it contains test commands, patterns, and conventions
2. Read the FULL implementation of the code you're going to test (see test-type-specific guidance below)
3. Understand how this code interacts with the rest of the codebase — what calls it, what it depends on, what assumptions it makes
4. Read existing tests in the same module — learn the patterns, the setup, the assertions
5. Read the test setup/helper files (`test/integration/setup/`, `test/e2e/setup/`) — understand what utilities and seed data are available
6. Think about what can go wrong. What are the edge cases? What happens with invalid input? What about unauthorized access? What about missing data?

If there is anything you can't determine from reading the code alone — domain knowledge, business rules, intended behavior that isn't obvious — add a TODO comment in the test file explaining the knowledge gap. We will fill these in later.

### What "read the full implementation" means per test type

**Unit tests (Go):**
- Read the function under test — its signature, input types, return types, branching logic
- Read existing `_test.go` files in the same package for patterns (table-driven tests, assertion style)

**Integration tests (Go backend):**
- Read the **handler** implementation in `features/`
- Read the **router configuration** in `main.go` to verify the exact endpoint path and HTTP method
- Read the **middleware chain** to understand what auth/authz/feature checks are applied
- Read `test/integration/setup/helpers.go` and `seed.go` for available test utilities and seed data
- Read `test/CLAUDE.md` if it exists — it contains module-specific testing conventions

**E2E tests (Playwright frontend):**
- Read the **Svelte page component** (`+page.svelte`) and its loader (`+page.ts`/`+page.server.ts`). Inventory all `data-testid` attributes, form element IDs, conditional rendering, and error states. Never guess selectors — they must match the actual DOM.
- Read the **backend code for all API endpoints called on the page** — understand what responses, errors, and status codes are possible
- Read `test/e2e/setup/helpers.ts`, `auth.ts`, and `seed.ts` for available test utilities

## Phase 4: Design test cases

Before writing code, list your test cases. For each one:

- **Scenario**: What situation are you testing?
- **Why it matters**: What bug or regression would this catch?
- **Expected behavior**: What should happen?

Discard any test case where the answer to "what bug would this catch?" is unclear.

## Phase 5: Implement

Write the tests. Follow the exact patterns from existing tests in the same module:

- Same file naming conventions
- Same imports and setup patterns
- Same assertion style
- Same test organization (subtests, table-driven, etc.)

After writing, run ONLY your new tests first to iterate quickly:

- **Unit tests (Go)**: `go test ./path/to/package -run TestName -v` from the module directory
- **Integration tests (Go)**: `make test-integration-single TEST=TestName` from the module directory (runs in Docker)
- **E2E tests (frontend)**: `npm run test-e2e -- --grep="test title pattern"` from the frontend directory (runs in Docker, grep filters by test title)

If tests fail, fix them. Iterate until they pass.

**Tests must be executed and passing before committing. No exceptions.** If the test runner fails for environmental reasons, stop and tell the user — do not commit unverified tests.

Then run `make check` / `npm run check` in the affected service(s) to ensure nothing else is broken.

## Phase 6: Finish

Commit your changes with a clear message describing what you tested and why.

Do NOT create a pull request. Just commit to the current branch.
