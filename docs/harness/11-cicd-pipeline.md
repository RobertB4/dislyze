# Workstream #11 — CI/CD Pipeline

## Current Setup

11 GitHub Actions workflows across three categories:

| Category | Workflows | Trigger |
|---|---|---|
| CI (lint + unit tests) | jirachi-ci, lugia-backend-ci, giratina-backend-ci, lugia-frontend-ci, giratina-frontend-ci | push (path-filtered) |
| Integration/E2E tests | lugia-backend-integration-tests, giratina-backend-integration-tests, lugia-frontend-e2e-tests | push (path-filtered) |
| Deploy | deploy-lugia, deploy-giratina, deploy-infrastructure | manual (workflow_dispatch) |

All CI workflows trigger on `push` only — **no `pull_request` triggers exist**. Each workflow is a single job with sequential steps and no cross-workflow dependencies.

### Go CI step order (lugia-backend, giratina-backend)

`go mod tidy` check → build → govulncheck → go vet → gosec → deadcode → golangci-lint → unit tests

### Frontend CI step order (lugia-frontend, giratina-frontend)

Build zoroark → npm ci → build → svelte-check → prettier + eslint → npm audit

### Integration/E2E tests

Fully Docker-based. Docker Compose spins up PostgreSQL, mock services, and the application. Tests run inside containers. E2E uses Playwright in a dedicated container with real Keycloak for SSO.

### Deployment

Manual trigger via `workflow_dispatch`. Builds Docker image → pushes to GCP Artifact Registry → runs vulnerability scan (blocks on CRITICAL/HIGH CVEs) → deploys to Cloud Run via Pulumi. GCP auth uses Workload Identity Federation (keyless OIDC).

## What's Already Agent-Friendly

### 1. Path-filtered triggers avoid unnecessary CI runs

Each workflow only runs when relevant paths change. A change to `lugia-backend/` doesn't trigger `giratina-frontend-ci`. Both backend CIs correctly trigger on `database/**` and `jirachi/**` changes (shared dependencies).

### 2. CI step ordering is logical

Within each workflow, the gates progress from fast/cheap to slow/expensive: module tidy check → build → static analysis → linting → unit tests. A formatting error fails before the test suite runs.

### 3. Vulnerability scanning at multiple levels

`govulncheck` catches Go dependency CVEs in CI. `npm audit --audit-level=high` catches npm CVEs. `gcloud artifacts docker images scan` catches container-level CVEs before deployment. Three layers of defense.

### 4. Deploy workflows have a security gate

Deployment is blocked if the vulnerability scan finds CRITICAL or HIGH CVEs. Emergency bypass is available but logged to Cloud Logging with a WARNING event — creating an audit trail.

### 5. Workload Identity Federation for GCP auth

No long-lived service account keys in secrets. OIDC-based keyless authentication is the current best practice for GitHub Actions → GCP.

## What's NOT Agent-Friendly

### 1. No `pull_request` trigger on any workflow (High Impact)

All CI runs on `push` only. This means:
- CI does not run when a PR is opened — only after code is pushed to a branch
- There are no required status checks on PRs (because no checks exist for PRs)
- An agent creating a PR has no CI feedback until after the push

In a harness environment, CI must run on PRs to give agents feedback before merge. Without this, agents push code, wait, see failures, push fixes, wait again — a slow feedback loop.

**Improvement strategy:** Add `pull_request` trigger to all CI workflows (alongside the existing `push` trigger). Then configure branch protection on `main` to require all CI checks to pass before merge.

### 2. No cross-workflow dependencies — tests run in parallel, not gated (High Impact)

Integration tests and CI (unit tests + lint) are separate workflows with no dependency. They run in parallel on every push. This means:
- Integration tests can run (and consume Docker resources) even when the code has a lint error
- E2E tests run even when unit tests fail
- There's no fail-fast behavior across workflows

The ideal gate ordering (`lint → unit → integration → e2e`) doesn't exist.

**Improvement strategy:** Two options:
1. **Merge into fewer workflows with job dependencies:** CI job runs first, then integration test job with `needs: ci`, then E2E job with `needs: integration`. This creates the gate ordering within a single workflow.
2. **Use `workflow_run` triggers:** Integration tests trigger only after CI succeeds. More modular but harder to reason about.

Option 1 is simpler and more agent-friendly — an agent reading one workflow file sees the entire pipeline.

### 3. Code generation (`make sqlc`) is not in CI (High Impact)

No CI step runs SQLC or validates that generated code is fresh. See [#7 Generation pipeline](07-generation-pipeline.md#2-no-ci-validation-that-generated-code-is-fresh-high-impact) for the full finding. This means an agent can modify SQL queries, commit stale generated code, and CI passes.

**Improvement strategy:** Add a generation freshness check as the first gate in Go CI workflows: `make sqlc && git diff --exit-code queries/`. If generated code is stale, CI fails immediately.

### 4. E2E tests trigger too broadly (Medium Impact)

`lugia-frontend-e2e-tests.yml` uses `paths-ignore: infrastructure/**` — it fires on any push that touches anything outside `infrastructure/`. A backend-only change to `giratina-backend/` triggers the full E2E suite (Docker Compose with 6 containers, Playwright). This wastes CI minutes and creates noise.

**Improvement strategy:** Change the E2E trigger to explicit path includes: `lugia-frontend/**`, `lugia-backend/**`, `zoroark/**`, `jirachi/**`, `database/**`. This matches the actual dependency graph of the E2E tests.

### 5. No CI-pass gate before deployment (Medium Impact)

Deploy workflows are manually triggered and do not check whether CI has passed for the commit being deployed. An agent (or human) could deploy code that hasn't passed lint, tests, or security checks.

**Improvement strategy:** Add a step at the start of deploy workflows that checks the commit's CI status via `gh api repos/{owner}/{repo}/commits/{sha}/check-runs` and fails if any required checks haven't passed.

### 6. Go tools installed with `@latest` — non-deterministic (Medium Impact)

Every CI run installs `govulncheck`, `gosec`, `gotestfmt`, and `deadcode` with `@latest`. This means:
- CI is not reproducible — the same code can produce different results on different days
- A new version of `gosec` could introduce new findings that break CI unexpectedly
- An agent debugging a CI failure can't reproduce it locally with certainty

**Improvement strategy:** Pin tool versions explicitly:
```yaml
- run: go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
- run: go install github.com/securego/gosec/v2/cmd/gosec@v2.22.4
```

Consider defining tool versions in a single place (e.g., a `tools.go` file or root Makefile) that CI reads.

### 7. No branch protection visible in the repository (Medium Impact)

No `CODEOWNERS`, no branch protection rulesets, no `dependabot.yml` in the repo. Branch protection may be configured in GitHub UI, but it's not codified or version-controlled.

**Improvement strategy:** Codify branch protection rules. At minimum, require:
- All CI checks passing before merge to `main`
- No direct pushes to `main` (force PR workflow)
- At least one approval (even if self-approval for a solo developer)

This can be done via GitHub API, Terraform, or Pulumi. Since the project already uses Pulumi for infrastructure, adding branch protection there would be natural.

### 8. GitHub Actions versions not SHA-pinned (Low Impact)

All actions use tag-based pinning (`@v4`, `@v7`). A compromised action tag could alter CI behavior. SHA pinning (e.g., `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683`) prevents this.

**Improvement strategy:** Pin critical actions to full SHA. At minimum: `actions/checkout`, `google-github-actions/auth`, and `pulumi/actions` (the actions with the highest blast radius). Use Dependabot or Renovate to keep SHA pins updated.

### 9. Pulumi version unpinned (Low Impact)

`pulumi-version: "latest"` in deploy workflows means infrastructure deploys use whatever Pulumi version is current at deploy time. A Pulumi version bump could change behavior or introduce breaking changes during a deployment.

**Improvement strategy:** Pin to a specific Pulumi version: `pulumi-version: "3.x.y"`. Update deliberately.

### 10. Docker base image inconsistency in production Dockerfiles (Low Impact)

`lugia-backend/Dockerfile` uses `alpine:3.22` (pinned). `giratina-backend/Dockerfile` uses `alpine:latest` (unpinned). This inconsistency means giratina's production image changes without code changes.

**Improvement strategy:** Pin both to the same Alpine version.

## Ideal Pipeline for Harness Engineering

The current pipeline vs. the ideal:

```
Current:
  [push] → CI (lint + unit) ──────────────────→ (no gate)
  [push] → Integration tests ─────────────────→ (no gate)
  [push] → E2E tests ─────────────────────────→ (no gate)
  [manual] → Deploy (no CI gate) ─────────────→ Production

Ideal:
  [push + PR] → Generate check ─→ Lint ─→ Typecheck ─→ Structural tests ─→ Unit tests
                                                                                 │
                                                                                 ▼
                                                                      Integration tests
                                                                                 │
                                                                                 ▼
                                                                           E2E tests
                                                                                 │
                                                                                 ▼
                                                                    All green → mergeable
                                                                                 │
                                                                                 ▼
                                                               [manual] Deploy (CI gate)
```

## Summary

| Finding | Impact | Action |
|---|---|---|
| No `pull_request` trigger | High | Add PR triggers, configure branch protection |
| No cross-workflow dependencies (no gate ordering) | High | Merge workflows or use `workflow_run` |
| Code generation not validated in CI | High | Add `make sqlc` + `git diff --exit-code` step |
| E2E tests trigger too broadly | Medium | Narrow path filters to actual dependencies |
| No CI-pass gate before deployment | Medium | Check commit status in deploy workflows |
| Go tools installed with `@latest` | Medium | Pin tool versions explicitly |
| No codified branch protection | Medium | Add via Pulumi or GitHub API |
| Actions not SHA-pinned | Low | SHA-pin critical actions |
| Pulumi version unpinned | Low | Pin to specific version |
| Docker base image inconsistency | Low | Pin both to same Alpine version |

The CI pipeline is functional — every important check exists somewhere. The main structural problem is that the checks run **in parallel with no gating**, and there's **no PR-level feedback loop**. An agent working in this environment gets CI results after pushing (not before merge) and has no guarantee that a passing lint check means integration tests will also pass. Adding PR triggers + gate ordering transforms the pipeline from "checks that run" to "gates that block."
