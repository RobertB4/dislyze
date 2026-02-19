# Workstream #10 — Structural Tests

## Current Setup

**Zero structural tests exist anywhere in the codebase.** All tests are behavioral — they test what the code does, not how the code is organized. Conventions around file naming, import direction, domain boundaries, and configuration consistency are followed by human discipline alone.

Go's module system provides one layer of structural protection: `jirachi` cannot import `lugia-backend` or `giratina-backend` (separate modules, no cross-dependency in `go.mod`). But intra-module structure is completely unenforced.

## What's Already Structurally Sound (by convention)

### 1. Feature domains don't cross-import

A search for cross-domain imports (`features/auth` importing `features/users`, etc.) returned zero results. Each feature domain is self-contained — it imports from `lib/`, `queries/`, and `jirachi/`, never from sibling feature domains.

### 2. Handler file pattern is uniform

Every feature domain has a `handler.go` with a `{Domain}Handler` struct and `New{Domain}Handler()` constructor. Individual endpoint files contain one public handler method each.

### 3. Configurations are consistent

`.golangci.yml`, `eslint.config.js`, `.prettierrc`, and `tsconfig.json` are identical across their respective modules (with only intentional differences in `sqlc.yaml` for lugia's type overrides).

### 4. SQL query files map to feature domains

Each `.sql` file in `queries_pregeneration/` is named after a feature domain: `auth.sql`, `users.sql`, `roles.sql`, `ip_whitelist.sql`. Strictly followed.

## What's NOT Agent-Friendly

### 1. No enforcement of feature domain isolation (High Impact)

Cross-domain imports don't exist today, but nothing prevents an agent from adding one. An agent implementing a feature in `features/auth/` that needs user data might import from `features/users/` directly instead of using the `queries` package or `lib/` utilities. This creates coupling between domains that's hard to untangle later.

**Improvement strategy:** Add a structural test that walks all Go source files under `features/{domain}/` and fails if any import matches `lugia/features/{other-domain}`. This is a simple AST walk — about 30 lines of Go. Alternatively, enforce via `depguard` in golangci-lint (as recommended in [#9](09-custom-linters.md)), but a dedicated structural test produces a clearer error message.

### 2. Enterprise feature flag sync is unvalidated (High Impact)

Adding a new enterprise feature requires coordinated changes across 4 files with string coupling (see [#3 Go backend](03-go-backend.md#1-enterprise-feature-flag-system-requires-changes-in-4-files-with-string-coupling-high-impact)). The strings in `jirachi/ctx/ctx.go` switch cases must match the constants in `lugia/lib/authz/enterprise_features.go`. If they diverge, the feature silently returns `false`.

Currently partially broken: `FeatureSSO` has no `RequireSSO()` middleware helper (SSO checks happen differently, via direct DB query in the handler).

**Improvement strategy:** Add a structural test that:
1. Creates a context with all enterprise features enabled
2. Calls `TenantHasFeature(ctx, feature)` for every `EnterpriseFeature` constant
3. Asserts all return `true`

If a new constant is added but the switch case in `ctx.go` is missing, the test fails. This catches the exact class of silent failure that makes this pattern dangerous.

### 3. Permission constants have no DB sync validation (High Impact)

`lugia/lib/authz/permissions.go` defines 4 resources × 2 actions = 8 permission combinations. `database/migrations/1_initial_schema.sql` inserts exactly 8 matching rows. Currently in sync — but adding a new permission requires both a Go constant AND a migration INSERT, with no validation that they match.

**Improvement strategy:** Add an integration test (requires DB) that:
1. Queries all rows from the `permissions` table
2. Compares against the Go `Resource` and `Action` constants
3. Fails if any Go constant lacks a DB row, or any DB row lacks a Go constant

This is an integration-level structural test (needs Docker), but it's the only reliable way to validate code-vs-schema sync without fragile SQL parsing.

### 4. `Validate()` method convention is unenforced (Medium Impact)

Every request body struct has a `Validate() error` method. This is the most important convention for input safety — without it, user input reaches handler logic unchecked. But there's no test verifying that every `*RequestBody` struct defines `Validate()`.

An agent adding a new endpoint might define a request body struct and skip the `Validate()` method entirely, or define it with the wrong signature.

**Improvement strategy:** Add a structural test that uses `go/ast` to:
1. Find all struct types matching `*RequestBody` in `features/` directories
2. Find all method declarations with receiver type matching those structs
3. Assert each struct has a `Validate() error` method

### 5. POST-for-all-mutations convention is unenforced (Medium Impact)

All mutation routes use `r.Post(...)` — never `r.Put`, `r.Patch`, or `r.Delete`. An agent trained on REST conventions will default to PUT/DELETE. Currently documented in CLAUDE.md but not enforced mechanically.

**Improvement strategy:** Add a structural test that parses `main.go`, finds all method calls on the Chi router, and fails if any use `Put`, `Patch`, or `Delete`. This is a simple grep-level check that could even be a shell script in CI.

### 6. Configuration drift has no detection (Medium Impact)

`.golangci.yml`, `eslint.config.js`, `.prettierrc` are identical across modules today. An agent modifying one module's config (e.g., adding a linter exclusion) creates drift that goes unnoticed. Over time, modules develop different rules, and agents encounter inconsistent behavior.

**Improvement strategy:** Two approaches:
1. **Single source of truth:** Move shared configs to the root and symlink or reference them from modules. This eliminates drift by design.
2. **Drift detection test:** A CI step that diffs the config files across modules and fails if they diverge (with an allowlist for intentional differences like `sqlc.yaml` overrides).

Option 1 is structurally cleaner. Option 2 is less disruptive to implement.

### 7. Route-to-auth-middleware mapping is unvalidated (Medium Impact)

`libctx.GetXxx()` functions panic if called outside the authenticated middleware chain (see [#3 Go backend](03-go-backend.md#2-context-getters-panic-on-missing-values-high-impact)). There's no validation that routes using these getters are actually inside the auth middleware group.

**Improvement strategy:** This is the hardest structural test to implement because it requires correlating router setup (in `main.go`) with handler implementations (in `features/*/`). Two options:
1. **Safe getter variants** (recommended in #3) — eliminates the panic risk entirely, making the structural test unnecessary
2. **Static analysis** — parse `main.go` to enumerate unauthenticated routes, then check those handler files don't call `libctx.GetXxx()`. Feasible but complex.

Option 1 is the better investment.

### 8. Seed data sync across Go/TypeScript/SQL is unvalidated (Low Impact)

`database/seed.sql`, `lugia-backend/test/integration/setup/seed.go`, and `lugia-frontend/test/e2e/setup/seed.ts` must stay in sync. See [#8 Testing infrastructure](08-testing-infrastructure.md#2-go-and-typescript-seed-data-are-manually-synced-high-impact) for the detailed finding. A structural test could validate that the same UUIDs and user counts appear in all three files, but this is better solved by generating the typed constants from the SQL.

## Priority Order for Implementation

The structural tests below are ordered by impact-to-effort ratio:

| Priority | Test | Effort | Impact |
|---|---|---|---|
| 1 | Enterprise feature flag sync | ~30 lines Go | Catches silent feature breakage |
| 2 | Feature domain isolation (no cross-imports) | ~30 lines Go (AST walk) | Prevents structural coupling |
| 3 | POST-only mutations | ~5 lines (grep in CI) | Prevents REST convention mistakes |
| 4 | Permission constants vs DB rows | ~40 lines Go (integration test) | Catches permission gaps |
| 5 | `Validate()` method on all RequestBody structs | ~40 lines Go (AST walk) | Prevents unvalidated input |
| 6 | Configuration drift detection | ~10 lines shell (diff in CI) | Prevents inconsistent linting |
| 7 | Route-to-middleware mapping | Complex (or solve with safe getters) | Prevents panics |

## Summary

| Finding | Impact | Action |
|---|---|---|
| No feature domain isolation enforcement | High | AST-based structural test for cross-imports |
| Enterprise feature flag sync unvalidated | High | Structural test with all-features-enabled context |
| Permission constants not validated against DB | High | Integration-level structural test |
| `Validate()` method convention unenforced | Medium | AST-based structural test |
| POST-only mutation convention unenforced | Medium | Grep-based CI check |
| Configuration drift undetected | Medium | Single source of truth or diff-based CI check |
| Route-to-middleware mapping unvalidated | Medium | Solve via safe getter variants instead |
| Seed data sync unvalidated | Low | Better solved by generation (see #8) |

Structural tests are the **enforcement backbone** of a harness engineering environment. Without them, conventions only hold as long as every agent reads and follows CLAUDE.md perfectly. With them, violations are caught mechanically before they reach the main branch. The good news: the codebase already follows its conventions consistently — the tests just need to lock that behavior in place.
