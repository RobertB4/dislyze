# Workstream #9 — Custom Linters

## Current Setup

Linting exists at two layers — Go and frontend — using off-the-shelf tools with minimal configuration. There are no custom linters, no teaching error messages, and no domain-specific enforcement rules anywhere.

### Go linting stack

| Tool | Config | What it catches |
|---|---|---|
| golangci-lint | `.golangci.yml` (identical in all 3 modules) | errcheck, staticcheck, ineffassign, unused, misspell, govet |
| go vet | Default set, no custom analyzers | Suspicious constructs, printf mismatches |
| gosec | CLI flags only, `queries/` excluded | OWASP security patterns (SQL injection, command injection, etc.) |
| govulncheck | Default | Known CVEs in dependencies |
| deadcode | lugia + giratina only, not jirachi | Unreachable functions |

golangci-lint enables 6 linters with zero custom configuration — no `linters-settings`, no `exclude-rules`, no `depguard`, no `revive`, no `importas`.

### Frontend linting stack

| Tool | Config | What it catches |
|---|---|---|
| ESLint | `eslint.config.js` (identical in all 3 projects) | JS/TS rules + Svelte plugin |
| Prettier | `.prettierrc` (identical in all 3 projects) | Formatting |
| svelte-check | tsconfig with `strict: true` | Type checking for Svelte components |

ESLint uses `typescript-eslint` with type-checked rules enabled, but **8 `no-unsafe-*` rules are globally disabled** — including `no-floating-promises`, `no-unsafe-assignment`, `no-explicit-any`, and all `no-unsafe-*` variants. This opts out of most of the value of type-checked linting.

### Pre-commit hooks

None. No `.husky/`, no `.pre-commit-config.yaml`, no active git hooks. Lint only runs in CI.

## What's Already Agent-Friendly

### 1. Linting configs are consistent across modules

All three Go modules share the same `.golangci.yml`. All three frontend projects share the same `eslint.config.js` and `.prettierrc`. An agent doesn't encounter different rules in different parts of the codebase.

### 2. CI gates lint before tests

Go CI runs: `go mod tidy` check → build → govulncheck → go vet → gosec → deadcode → golangci-lint → tests. Frontend CI runs: build → svelte-check → prettier + eslint → npm audit. Lint failures are caught before slower test steps.

### 3. TypeScript strict mode is enabled

All frontend `tsconfig.json` files use `"strict": true`, enabling `strictNullChecks`, `noImplicitAny`, and other strictness flags. `svelte-check` enforces these in Svelte component `<script>` blocks.

### 4. Formatting is fully automated

Prettier handles all formatting decisions. No style debates, no inconsistency. An agent running `npx prettier --write .` produces correct formatting every time.

## What's NOT Agent-Friendly

### 1. No teaching error messages anywhere (High Impact)

This is the core gap for harness engineering. Every lint error an agent encounters is a generic message from an off-the-shelf tool:

- `errcheck`: "Error return value of X is not checked" — doesn't say how to handle it in this codebase
- `staticcheck SA1019`: "X has been deprecated" — doesn't say what to use instead in this codebase
- `svelte-check`: "Type 'X' is not assignable to type 'Y'" — doesn't explain the codebase's pattern

In a harnessed environment, lint errors should teach the fix. For example: "Error return value not checked. In this codebase, wrap errors with `errlib.New(err, statusCode, userMessage)` — see [link to CLAUDE.md pattern]."

**Improvement strategy:** This is a phased effort:
1. **Phase 1 (documentation):** For the most common lint errors agents will encounter, add a "common lint errors and how to fix them" section to CLAUDE.md. This doesn't require custom linters — it just pre-teaches the fixes.
2. **Phase 2 (custom rules):** Add custom golangci-lint rules via `revive` or custom `analysis.Analyzer` implementations for codebase-specific patterns (e.g., "always use `errlib.New`, never `fmt.Errorf` in handlers").
3. **Phase 3 (teaching messages):** Configure custom error messages on rules that support it (e.g., `revive` rules accept custom messages).

### 2. No import direction enforcement (High Impact)

There is no `depguard`, `importas`, or any mechanism preventing wrong-direction imports:

- **Cross-module:** Go's module system prevents `jirachi` from importing `lugia-backend` (they're separate modules). This is protected mechanically.
- **Intra-module:** Nothing prevents `lib/` from importing `features/`, `features/auth` from importing `features/users`, or handlers from directly importing `queries_pregeneration/`. These dependency direction violations are invisible.

An agent adding a new feature might import from another feature domain or create a circular dependency within a module that only manifests as subtle bugs or maintenance problems later.

**Improvement strategy:** Add `depguard` to golangci-lint configuration with rules like:
- `features/*` packages may import from `lib/*` but NOT from other `features/*` packages
- `lib/*` packages may NOT import from `features/*`
- Application code may NOT import from `queries_pregeneration/` (only SQLC config references it)

This enforces the dependency direction that currently only exists as an implicit convention.

### 3. TypeScript type safety heavily opted out (High Impact)

Eight `no-unsafe-*` rules are globally disabled in all frontend ESLint configs:

```javascript
'@typescript-eslint/no-floating-promises': 'off',
'@typescript-eslint/no-unsafe-assignment': 'off',
'@typescript-eslint/no-unsafe-member-access': 'off',
'@typescript-eslint/no-unsafe-call': 'off',
'@typescript-eslint/no-unsafe-argument': 'off',
'@typescript-eslint/no-unsafe-return': 'off',
'@typescript-eslint/no-explicit-any': 'off',
'@typescript-eslint/no-redundant-type-constituents': 'off',
```

`no-floating-promises` being off is particularly dangerous — fire-and-forget async bugs are completely silent. `no-explicit-any` being off means agents can use `any` freely, defeating TypeScript's purpose.

These rules were likely disabled because the codebase has existing violations that would be noisy to fix. But for agents, this removes guardrails that would catch real bugs.

**Improvement strategy:** Re-enable rules incrementally:
1. Start with `no-floating-promises` (highest bug risk) — fix existing violations, then enable as `error`
2. Enable `no-explicit-any` as `warn` first to surface violations without blocking CI
3. Tackle `no-unsafe-*` rules last as they likely require fixing the hand-written API types (which the OpenAPI contract layer from [#5](05-openapi-contract-layer.md) would solve)

### 4. No pre-commit hooks (Medium Impact)

Lint only runs in CI. An agent can commit and push code that fails formatting, type checking, or lint rules, then wait for CI to report the failure. The feedback loop is: commit → push → wait ~2 min → read CI output → fix → repeat.

With pre-commit hooks, the agent gets immediate feedback before the commit is even created.

**Improvement strategy:** Add a lightweight pre-commit hook (via Husky or simple git hooks) that runs:
- `prettier --check` on staged files
- `svelte-check` for frontend changes
- `golangci-lint run` for Go changes

Keep the hook fast (< 10 seconds) to avoid disrupting the agent's flow. The full lint suite still runs in CI as the authoritative gate.

### 5. No codebase-specific lint rules (Medium Impact)

Rules like these exist as implicit conventions but have zero enforcement:

- "All mutations use POST, never PUT/PATCH/DELETE" — no lint rule checks HTTP method registration
- "Never use `$effect` without exhausting alternatives" — no ESLint rule flags `$effect` usage
- "Never call `libctx.GetXxx` outside authenticated routes" — no static analysis validates this
- "Always use `errlib.New` in handlers, never raw `fmt.Errorf`" — no rule enforces this
- "Felte values use `$` prefix in templates" — no Svelte-specific rule checks this

Each of these is a pit an agent will fall into. Custom lint rules turn these pits into guardrails.

**Improvement strategy:** Prioritize rules by frequency of agent error:
1. `$effect` usage warning (ESLint custom rule or Svelte compiler config)
2. `errlib.New` enforcement in handler files (revive or custom analyzer)
3. HTTP method convention (custom analyzer that checks Chi router registration)

These don't need to be perfect on day one — even a `warn`-level rule that says "Are you sure about this?" is better than silence.

### 6. golangci-lint runs `--fast-only` in CI (Low Impact)

The GitHub Actions golangci-lint step uses `--fast-only`, which may skip slower but more powerful checks (like full `staticcheck` analysis) depending on cache state. This means an agent might pass CI on one run but fail on another with the same code.

**Improvement strategy:** Remove `--fast-only` and accept the longer CI time. The 5-minute timeout is already configured in the config file. Alternatively, ensure the CI cache is warm by running the full lint in a separate scheduled workflow.

### 7. Jirachi is under-linted compared to backends (Low Impact)

Jirachi has no `deadcode` check in its Makefile or CI, and no `lint` Makefile target. The CI pipeline is shorter than the backends (no build step, no deadcode). Since jirachi is the shared library used by both backends, it arguably deserves the strictest linting.

**Improvement strategy:** Add `lint` and `deadcode` targets to jirachi's Makefile. Add the corresponding CI steps.

### 8. No root-level lint command (Low Impact)

The root Makefile has no `lint` target. An agent wanting to lint the entire codebase must run lint in each module independently.

**Improvement strategy:** Add a root `make lint` target (same pattern as the proposed `make generate` from [#7](07-generation-pipeline.md)). This is part of the broader "single command for everything" theme.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No teaching error messages | High | Phase 1: document fixes in CLAUDE.md. Phase 2: custom rules with messages |
| No import direction enforcement | High | Add `depguard` rules to golangci-lint |
| TypeScript type safety opted out (8 rules disabled) | High | Re-enable incrementally, starting with `no-floating-promises` |
| No pre-commit hooks | Medium | Add lightweight Husky hooks for formatting + lint |
| No codebase-specific lint rules | Medium | Add custom rules for top agent-error patterns |
| golangci-lint `--fast-only` in CI | Low | Remove flag for deterministic results |
| Jirachi under-linted | Low | Add lint + deadcode to jirachi CI |
| No root-level lint command | Low | Add `make lint` at root |

The linting infrastructure is functional but **entirely passive** — it catches generic code quality issues but doesn't teach agents how this specific codebase works. The harness engineering ideal is lint rules that act as guardrails with explanations, turning every mistake into a learning moment. The highest-leverage improvements are **adding `depguard` for import direction** (prevents structural mistakes mechanically) and **re-enabling TypeScript unsafe rules** (prevents type safety erosion). Teaching error messages are the long-term goal but require custom lint rule development.
