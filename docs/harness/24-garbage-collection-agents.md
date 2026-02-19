# Workstream #24 — Garbage Collection Agents

## Current Setup

Entropy is fought reactively — humans notice dead code, stale docs, or unused dependencies and clean them up manually. A handful of automated tools exist (Go dead code detection, vulnerability scanning, dependency hygiene checks), but they run only on push and cover only a subset of entropy types. There are no periodic cleanup processes, no scheduled scans, and no agent role dedicated to fighting entropy.

### What exists

| Tool | What it catches | Trigger | Coverage |
|---|---|---|---|
| `deadcode` (Go) | Unreachable functions | Per-push CI | lugia-backend, giratina-backend (not jirachi) |
| `golangci-lint unused` | Locally unused identifiers | Per-push CI | All three Go modules |
| `golangci-lint ineffassign` | Variables written but never read | Per-push CI | All three Go modules |
| `staticcheck` | Deprecated API usage | Per-push CI | All three Go modules |
| `govulncheck` | Known CVEs in Go dependencies | Per-push CI | All three Go modules |
| `npm audit` | Known CVEs in npm packages | Per-push CI | lugia-frontend, giratina-frontend (not zoroark) |
| `go mod tidy` diff check | Unused Go dependencies | Per-push CI | All three Go modules |
| `/selfimprove` | Stale CLAUDE.md content | Manual invocation | All CLAUDE.md files (2 missing) |

### Evidence of manual cleanup

Git history shows reactive cleanup — humans spotting and removing dead code:

```
d8f23a8 remove dead code
3533fe6 remove dead code
31e8cfb remove unneeded queries
e3762d1 remove unneeded dependencies
468a472 delete unused playwright config
b2ac262 remove schema.go files
```

No commits suggest automated or scheduled cleanup. All cleanup is ad-hoc.

## What's Already Agent-Friendly

### 1. Go dead code detection is automated and gating

`make deadcode` runs in CI for both backends. If dead code is introduced, CI fails. This is the gold standard for garbage collection — automated detection with a hard gate. The tool (`golang.org/x/tools/cmd/deadcode`) performs whole-program reachability analysis, catching functions that are exported but never called.

### 2. Dependency hygiene is enforced in CI

`go mod tidy` runs in CI with a diff check — if it would modify `go.mod` or `go.sum`, CI fails. This means unused Go dependencies are caught automatically. Combined with `govulncheck` and `npm audit`, the dependency layer is well-covered.

### 3. Go compiler prevents unused import accumulation

Go refuses to compile with unused imports. This is a language-level guarantee — there is zero unused import debt possible. No tool needed.

### 4. Zero TODO/FIXME comments exist

The codebase has no orphaned TODO, FIXME, HACK, or XXX comments. This is a form of cleanliness that prevents a common entropy vector. Agents don't encounter abandoned work items embedded in code.

### 5. `/selfimprove` is a working doc-gardening mechanism

While manually triggered and imperfect (see [#22 Agent retrospective](22-agent-retrospective.md)), `/selfimprove` is a functional tool for identifying and fixing stale CLAUDE.md content. It has been exercised once successfully.

## What's NOT Agent-Friendly

### 1. No periodic/scheduled cleanup scans (High Impact)

All existing tools run only on push — meaning they only check code that was recently modified. A vulnerability introduced in a dependency three months ago, in a path not touched by recent commits, sits undetected until someone happens to modify that path.

Similarly, `deadcode` only catches dead code when the file containing the caller is pushed. If a handler is deleted but the query it called remains in a different file that wasn't touched, the dead query persists.

**Improvement strategy:** Add a scheduled CI workflow that runs weekly:

```yaml
name: Weekly Cleanup Scan
on:
  schedule:
    - cron: '0 9 * * 1'  # Monday 9am UTC
  workflow_dispatch: {}

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Dead code scan (all modules)
        run: |
          cd lugia-backend && make deadcode
          cd ../giratina-backend && make deadcode
          cd ../jirachi && make deadcode  # requires adding deadcode to jirachi Makefile
      - name: Vulnerability scan
        run: |
          govulncheck ./lugia-backend/...
          govulncheck ./giratina-backend/...
          govulncheck ./jirachi/...
          cd lugia-frontend && npm audit --audit-level=high
          cd ../giratina-frontend && npm audit --audit-level=high
          cd ../zoroark && npm audit --audit-level=high
```

This ensures nothing rots between pushes. The `workflow_dispatch` trigger allows manual runs too.

### 2. No dead code detection for TypeScript/Svelte (High Impact)

The Go side has `deadcode` + `unused` linter. The TypeScript/Svelte side has nothing. Dead exports in zoroark (the shared component library), unused route helpers in frontends, and orphaned utility functions accumulate silently.

This matters especially for zoroark — as a component library consumed by two frontends, it's easy for a component to become unused in both consumers without anyone noticing.

**Improvement strategy:** Add a TypeScript dead code detection tool. Options:

- **`knip`** — purpose-built for finding unused files, exports, dependencies, and types in TypeScript/JavaScript projects. Covers the full entropy surface.
- **`ts-prune`** — focused specifically on unused exports. Lighter weight.

Either tool can be added as a CI step and a Makefile target. `knip` is the more comprehensive choice — it also catches unused dependencies, which `npm audit` doesn't cover.

### 3. No orphaned SQL query detection (Medium Impact)

`queries_pregeneration/` directories contain SQL files that SQLC generates into Go code. When a handler that called a generated query is deleted, the SQL file and its generated Go code remain. `deadcode` doesn't catch this because SQLC generates an interface method that's "used" by the interface even if no handler calls it.

Over time, orphaned queries accumulate — SQL files, generated Go code, and Querier interface methods that no one calls.

**Improvement strategy:** A structural test (from [#10 Structural tests](10-structural-tests.md)) that cross-references generated Querier method names against actual handler code. If a Querier method isn't called from any handler (or middleware, or other non-test code), it's a candidate for removal.

Alternatively, a garbage collection script that: lists all Querier interface methods → greps for their usage in non-generated, non-test Go files → reports unmatched methods.

### 4. No CLAUDE.md freshness detection (Medium Impact)

CLAUDE.md files can become stale as the codebase evolves — documented patterns that no longer exist, referenced files that moved, conventions that changed. Currently, staleness is only detected when `/selfimprove` is manually invoked or when a human reads the file and notices the drift.

Known stale content:
- `giratina-backend/CLAUDE.md` lists "Next Steps" that are already implemented
- Root `README.md` describes a two-component project (now six)
- `/selfimprove` references `jirachi/CLAUDE.md` and `zoroark/CLAUDE.md` which don't exist

**Improvement strategy:** A structural test that validates CLAUDE.md references:
- Every file path mentioned in a CLAUDE.md exists
- Every Makefile target mentioned is real
- Every command mentioned works

This is a mechanical freshness check — it doesn't validate whether the prose is accurate, but it catches broken references. For prose freshness, the `/selfimprove` loop (with improved abstraction level from [#22](22-agent-retrospective.md)) is the right mechanism.

### 5. Seed data sync has no validation (Medium Impact)

Three separate seed data files must stay in sync manually:
- `database/seed.sql` (822 lines)
- `lugia-backend/test/integration/setup/seed.go` (539 lines)
- `lugia-frontend/test/e2e/setup/seed.ts` (381 lines)

When one is updated and the others aren't, tests may pass individually but fail when run together, or test against stale data that doesn't match the application's expectations.

**Improvement strategy:** A structural test that validates key invariants across seed files: same tenant UUIDs, same user UUIDs, same role names, same permission sets. This was proposed in [#10 Structural tests](10-structural-tests.md) and [#8 Testing infrastructure](08-testing-infrastructure.md).

### 6. No configuration drift detection between modules (Low Impact)

The three Go modules have identical `.golangci.yml` files. The two frontends have similar ESLint and Prettier configs. If one module's config is updated and the others aren't, enforcement becomes inconsistent — an agent passing lint in one module might fail in another for the same code.

**Improvement strategy:** A structural test that diffs configuration files across modules and flags divergence. Or, consolidate configs by using shared config files (golangci-lint supports config inheritance; ESLint supports `extends`).

### 7. No stale branch cleanup (Low Impact)

Three remote branches exist besides `main`: `add-pulumi-infra`, `shared-library`, `staging`. The first two appear abandoned (merged or superseded). Without cleanup, the branch list grows and confuses agents running `git branch -r`.

**Improvement strategy:** Delete merged/abandoned branches. Configure GitHub to auto-delete branches after PR merge. This is a one-time cleanup plus a setting change.

## Types of Garbage Collection Agents

Based on the entropy vectors in this codebase, three agent roles would provide the most value:

### 1. Code Hygiene Agent (highest priority)

Runs periodically or on-demand. Checks:
- Dead Go code (`deadcode` across all modules)
- Dead TypeScript/Svelte exports (`knip` or `ts-prune`)
- Orphaned SQL queries (cross-reference Querier methods against handler usage)
- Unused dependencies beyond what `go mod tidy` catches
- Stale remote branches

Could be implemented as a scheduled CI workflow plus a `/cleanup-scan` command.

### 2. Doc Gardening Agent (medium priority)

Extends `/selfimprove` with:
- CLAUDE.md reference validation (file paths, commands, targets all exist)
- README freshness check (component list matches actual directory structure)
- Harness audit doc currency (are findings still accurate after implementation?)

Could be implemented as a structural test suite plus an enhanced `/selfimprove` that reads improvement proposals from [#22](22-agent-retrospective.md).

### 3. Quality Grading Agent (lower priority)

Periodically assesses code quality across feature domains:
- Convention adherence (does code follow CLAUDE.md patterns?)
- Type safety trend (are TypeScript `any` usages growing?)
- Test coverage per feature domain
- Technical debt hotspots (files with most churn, most fix commits)

This requires metrics infrastructure from [#23 Observability](23-observability-metrics.md) before it can operate.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No periodic/scheduled cleanup scans | High | Add weekly scheduled CI workflow for deadcode + vulnerability scans |
| No dead code detection for TypeScript/Svelte | High | Add `knip` or `ts-prune` for unused export detection |
| No orphaned SQL query detection | Medium | Structural test cross-referencing Querier methods against handlers |
| No CLAUDE.md freshness detection | Medium | Structural test validating file paths and commands referenced in CLAUDE.md |
| Seed data sync has no validation | Medium | Structural test comparing key invariants across seed files |
| No configuration drift detection between modules | Low | Structural test diffing configs, or consolidate to shared configs |
| No stale branch cleanup | Low | Delete abandoned branches, enable auto-delete on merge |

Garbage collection is what prevents the harness from degrading over time. The existing Go dead code detection and dependency hygiene tools are a strong foundation. The gaps are on the TypeScript side (no dead export detection), the SQL layer (orphaned queries), and the documentation layer (stale CLAUDE.md content). The most impactful addition is a weekly scheduled scan — it catches entropy that per-push CI misses because the rotting code lives in paths no one recently touched.
