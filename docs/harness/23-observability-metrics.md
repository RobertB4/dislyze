# Workstream #23 — Observability & Metrics

## Current Setup

No agent-effectiveness metrics are collected or generated. The infrastructure to measure agent performance does not exist. The only observable signals are: CI pass/fail (visible in GitHub UI), git commit history (queryable but not analyzed), and production service health monitoring (GCP Cloud Monitoring — measures deployed services, not development quality).

### What exists

| Signal | Source | Preserved? |
|---|---|---|
| CI pass/fail per workflow | GitHub Actions | Yes — GitHub UI/API |
| Per-step CI timing | GitHub Actions | GitHub UI only — not queryable |
| Go test pass/fail + timing | `go test -json` piped to `gotestfmt` | No — JSON stream is consumed and lost |
| Playwright HTML test report | Generated in `test-results/` | No — never uploaded as artifact |
| Code coverage | Not collected | No — no `-cover` flag used |
| Git commit history | Git | Yes — but not analyzed |
| Production health | GCP Cloud Monitoring (uptime, latency, errors) | Yes — but measures deployed services, not agent output |

The key observation: `go test -json` already produces machine-readable output with per-test names, durations, and pass/fail. This data is being piped through `gotestfmt` for human-readable display and then discarded. Structured test data exists in the pipeline — it's just not preserved.

## What's Already Agent-Friendly

### 1. Go test `-json` flag is already in use

Every `test-unit` target across all three Go modules runs `go test -json`. The structured data is generated — it just needs to be tee'd to a file before being consumed by `gotestfmt`. This is the lowest-effort, highest-value metric to capture.

### 2. Playwright supports multiple reporters

The E2E config already uses two reporters (`html` and `list`). Adding a JUnit XML or JSON reporter is a one-line change. Playwright's built-in reporters produce per-test timing and pass/fail data that's immediately useful.

### 3. GitHub Actions API provides workflow run data

Once CI runs on PRs (workstream #11), GitHub's API provides: workflow run duration, per-job timing, billable minutes, and run status — all queryable without any instrumentation changes. This is free infrastructure.

### 4. Production monitoring is mature

GCP Cloud Monitoring covers: uptime checks, CPU/memory/latency/error rate alerts, Cloud Run logging with 1-year retention. While this doesn't measure agent effectiveness, it measures the downstream impact of code quality — a slow rollback or incident could be traced back to the PR that caused it.

## What's NOT Agent-Friendly

### 1. Structured test output is generated but discarded (High Impact)

`go test -json` produces per-test timing, pass/fail, and package information — exactly the data needed for tracking test reliability and performance over time. But it's piped directly to `gotestfmt` and lost.

Similarly, Playwright generates an HTML report but it's never uploaded as a CI artifact. After the job completes, the report is gone.

**Improvement strategy:** Preserve structured test output with minimal changes:

For Go tests — tee the JSON stream:
```makefile
test-unit:
    go test `go list ./... | grep -v test` -json -v 2>&1 | tee test-output.json | gotestfmt -hide=empty-packages,successful-tests
```

For Playwright — add a JUnit XML reporter:
```javascript
reporter: [
    ["html", { outputFolder: "./test-results", open: "never" }],
    ["list"],
    ["junit", { outputFile: "test-results/results.xml" }]
],
```

Then upload both as CI artifacts using `actions/upload-artifact`. This gives queryable per-test data for every CI run with no change to the developer experience.

### 2. No code coverage collection (High Impact)

No Go test command uses `-cover` or `-coverprofile`. No frontend coverage tool is configured. An agent has no way to know if their new code is covered by tests, and there's no gate preventing coverage regression.

This was also flagged in [#8 Testing infrastructure](08-testing-infrastructure.md).

**Improvement strategy:** Add coverage to Go test commands:

```makefile
test-unit:
    go test `go list ./... | grep -v test` -json -v -cover -coverprofile=coverage.out 2>&1 | tee test-output.json | gotestfmt -hide=empty-packages,successful-tests
```

Add a CI step that reports coverage: `go tool cover -func coverage.out`. Upload `coverage.out` as an artifact. Consider a coverage threshold gate (e.g., no PR can decrease total coverage) once there's enough test coverage to make the threshold meaningful.

### 3. PR-based metrics don't exist because PRs don't exist (High Impact)

The most valuable agent-effectiveness metrics are PR-lifecycle metrics:
- **Review rounds** — how many request-changes cycles before approval?
- **Time-to-merge** — how long from PR open to merge?
- **CI check results per PR** — how often does CI fail on first push?
- **Iterations before green** — how many fix-up commits before CI passes?

With only 3 PRs in the repository's history (all with empty bodies, zero reviews, self-merged immediately), there is no data to measure. This is the most fundamental blocker: the workflow must produce PRs before PR metrics can exist.

**Improvement strategy:** This is entirely dependent on adopting the PR-based workflow from [#15 Version control](15-version-control-strategy.md) and [#18 Workflow](18-agent-workflow-orchestration.md). Once agents work on branches and open PRs, all of these metrics become automatically available through the GitHub API.

The prerequisite chain:
```
PR triggers on CI (#11) → branch protection (#15) → PR-based workflow (#15, #18)
    → PR lifecycle data exists → metrics become queryable
```

### 4. No way to distinguish agent-authored commits from human-authored commits (Medium Impact)

Git history doesn't indicate who (or what) authored each commit. An agent commit looks identical to a human commit. This makes it impossible to measure agent-specific metrics — agent success rate, agent rework rate, agent vs. human quality.

**Improvement strategy:** Convention-based attribution. Two approaches:

1. **Co-author trailer** — Claude Code already adds `Co-Authored-By: Claude` to commits. If this convention is consistently applied, `git log --grep="Co-Authored-By: Claude"` identifies agent-authored commits.
2. **Branch naming convention** — if agent branches use a prefix like `agent/feat/...`, they're filterable by branch name in PR history.

Neither requires tooling changes — just consistent conventions documented in CLAUDE.md.

### 5. No dashboard or reporting mechanism (Medium Impact)

Even if metrics were collected, there's nowhere to view them. No dashboard, no periodic report, no script that summarizes agent performance.

**Improvement strategy:** Start with a simple script before investing in dashboards. A `scripts/metrics.sh` that queries:
- `gh api repos/{owner}/{repo}/actions/runs` for CI pass/fail rates
- `gh pr list --state all --json` for PR lifecycle data
- `git log` for commit patterns

This is a periodic tool, not a real-time dashboard. Run it weekly or monthly to track trends. A dashboard becomes worthwhile only when the data volume justifies it.

### 6. No structural test failure tracking (Low Impact)

When structural tests are implemented (workstream #10), their failure patterns over time would indicate whether agents are learning conventions or repeatedly violating them. Currently structural tests don't exist, so there's nothing to track.

**Improvement strategy:** Defer until structural tests are implemented. When they are, ensure their output is captured in the same structured test output mechanism (Go test JSON).

## Metrics That Matter for Agent-First Development

Prioritized by what they reveal about agent effectiveness:

| Metric | What it measures | Prerequisite |
|---|---|---|
| CI pass rate on first push | Does the agent produce code that passes CI on the first try? | PR workflow + CI on pull_request |
| Fix-up commits per PR | How many iterations before CI passes? | PR workflow |
| Review rounds per PR | How much back-and-forth before approval? | PR workflow + review process |
| Time-to-merge | End-to-end efficiency from task start to merge | PR workflow |
| Test coverage trend | Is coverage increasing, stable, or declining? | Coverage collection in CI |
| Structural test failures | Are agents following architectural conventions? | Structural tests (workstream #10) |
| Retrospective observation frequency | What do agents consistently struggle with? | `/retro` command (workstream #22) |

### What's measurable today (no workflow changes needed)

These can be computed from the existing git history:

| Metric | Current value | What it suggests |
|---|---|---|
| Fix-commit ratio | ~17% (93 of ~542 commits) | Moderate rework rate |
| CI-fix commit frequency | ~15 commits | CI configuration causes friction |
| Commit message convention adherence | ~30% use conventional prefixes | Inconsistent, no enforcement |

These are rough proxies — without PR data or agent attribution, they measure the project, not the agent.

## Summary

| Finding | Impact | Action |
|---|---|---|
| Structured test output generated but discarded | High | Tee Go test JSON, add Playwright JUnit XML, upload as CI artifacts |
| No code coverage collection | High | Add `-cover -coverprofile` to Go tests |
| PR metrics don't exist because PRs don't exist | High | Adopt PR-based workflow (prerequisite from workstreams #11, #15, #18) |
| No agent vs. human commit attribution | Medium | Convention-based: Co-Author trailer, branch naming prefix |
| No dashboard or reporting mechanism | Medium | Start with a simple query script, dashboard later |
| No structural test failure tracking | Low | Defer until structural tests exist (workstream #10) |

Observability is the workstream that depends on almost everything else. Without PRs, there are no PR metrics. Without coverage, there's no coverage trend. Without structural tests, there's no convention adherence data. Without the retrospective loop, there's no qualitative feedback. The right sequence: build the workflow (PRs, CI gates, test coverage), then wire up measurement. The lowest-hanging fruit — preserving the Go test JSON output that's already being generated — costs almost nothing and should be done immediately as part of any CI improvement work.
