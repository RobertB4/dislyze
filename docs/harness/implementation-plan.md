# Harness Implementation Plan

This document is the prioritized implementation plan derived from the 24 workstream audits. It defines what to build, in what order, and how to measure whether improvements actually help.

## Guiding Principles

1. **Measure first** — without data, we're guessing. Every improvement should be measurable.
2. **Quick wins first** — high impact, low effort items ship before anything else.
3. **High impact before medium** — but low-effort medium-impact items can jump the queue.
4. **Flexibility** — this plan will evolve. Adjust as we learn from real agent sessions.

---

## Workflow

### Bootstrapping phase (Tier 0 + Tier 1)

During Tier 0 and Tier 1, we're building the workflow infrastructure itself. We can't use branch protection before we set it up.

- **Commit directly to main** — these are configuration/documentation changes, not application code
- **Human handles pushing** — the agent does not push until branch protection is in place
- **Follow the steady-state workflow as much as possible** — use `/plan`, `/implement`, `make verify`, self-review, etc. as soon as each becomes available. Skip steps that aren't feasible yet.

### Steady-state workflow (Tier 2 onwards, and all future work)

Once branch protection and PR infrastructure are in place, every task follows this flow:

```
1. INTAKE        Human defines task or agent picks from backlog
2. PLAN          /plan — research, multiple options, clarifying questions
3. SCOPE         Define in/out of scope, acceptance criteria
4. IMPLEMENT     /implement — code on a feature branch
5. VERIFY        make verify — lint + typecheck + unit tests
6. SELF-REVIEW   Agent reviews its own diff before committing
7. DELIVER       Commit to branch, open PR using the template
8. REVIEW        Human reviews (or /review for agent review)
9. MERGE         Squash-merge to main after CI passes
10. SCORE        Add session to sessions.js, check dashboard
11. REFLECT      Optionally /retro or /selfimprove
```

Steps are adopted progressively — each one is used as soon as the infrastructure for it exists. The agent should not push to remote until branch protection is enforced on main.

---

## Tier 0 — Measurement Foundation

**Goal:** Before changing anything, establish a way to measure agent effectiveness so we can prove improvements work.

### 0.1 Define session scoring rubric

Create a scoring system applied after each agent session (5 dimensions, 1-5 scale):

| Dimension | 1 (poor) | 3 (acceptable) | 5 (excellent) |
|---|---|---|---|
| Task completion | Didn't finish or fundamentally wrong | Finished with notable issues | Clean, correct completion |
| Convention adherence | Violated established patterns | Mostly followed, minor drift | Perfectly followed conventions |
| First-try CI pass | 3+ fix-up rounds | One fix needed | Passed on first push |
| Scope discipline | Significant unrelated changes | Minor scope drift | Exactly scoped to task |
| Self-sufficiency | Needed constant course-correction | Some human guidance needed | Worked autonomously to completion |

Additionally, record:

| Field | Description |
|---|---|
| Task difficulty | 1-5 scale: trivial fix → multi-file feature → cross-cutting architectural change |
| Task type | bug-fix, feature, refactor, test, docs, infra |
| Conversation turns | Number of human↔agent back-and-forth rounds |
| Session duration | Approximate wall-clock time |
| Harness version | Which tier of improvements were in place (baseline, tier-1, tier-2, etc.) |

The **task difficulty** dimension is critical — it lets us see at which complexity level agents start to struggle, and whether harness improvements raise that threshold over time.

**Effort:** ~15 min to define, formalize below in the data structure.

### 0.2 Create tracking data structure

Create `docs/harness/metrics/sessions.js`:

```json
{
  "sessions": [
    {
      "date": "2025-06-20",
      "task": "Add IP whitelist enterprise feature",
      "type": "feature",
      "difficulty": 4,
      "harness_version": "baseline",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 3,
        "scope": 4,
        "self_sufficiency": 3
      },
      "turns": 12,
      "duration_minutes": 45,
      "notes": "Agent struggled with middleware ordering, produced concrete selfimprove output"
    }
  ]
}
```

**Effort:** ~10 min.

### 0.3 Create metrics dashboard

Create `docs/harness/metrics/index.html` — a single HTML file with Chart.js (no build step, no TypeScript, no framework). Reads `sessions.js` and renders:

1. **Overall score trend** — average score per dimension over time, grouped by harness version
2. **Score by difficulty** — radar chart or grouped bar showing scores at each difficulty level
3. **Difficulty threshold** — at what difficulty level does average score drop below 3 (acceptable)?
4. **Turns per task** — does conversation length decrease over time?
5. **Score by task type** — which task types do agents handle best/worst?

Open it with `open docs/harness/metrics/index.html` — no server needed.

**Effort:** ~1 hour.

### 0.4 Score baseline sessions retroactively

Review recent agent sessions from memory and score them. We need at least 3-5 baseline data points. Sessions to consider:

- IP whitelist feature implementation
- SSO tenant creation from giratina
- Any recent feature work or bug fixes

These retroactive scores won't be perfectly accurate, but they establish the "before" in our before/after comparison.

**Effort:** ~15 min.

### Tier 0 exit criteria
- Scoring rubric defined ✓ (in this document)
- `sessions.js` created with baseline data
- `index.html` dashboard renders charts from the data
- We can answer: "what is the current average agent score at difficulty level 3?"

---

## Tier 1 — Quick Wins (High Impact, Low Effort)

**Goal:** Ship all high-impact, low-effort improvements in one batch. Each item should take ≤30 minutes.

### Makefile improvements

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 1.1 | `make verify` in root Makefile | #20 | ~10 min | Single command to lint + typecheck + unit test across entire monorepo |
| 1.2 | `make generate` in root Makefile | #7, #20 | ~5 min | Single command to regenerate all SQLC across all modules |

### CI/CD improvements

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 1.3 | Add `pull_request` triggers to all CI workflows | #11 | ~15 min | CI runs on PRs → first-push pass rate becomes measurable |
| 1.4 | PR template (`.github/PULL_REQUEST_TEMPLATE.md`) | #19 | ~10 min | Agents know what to include in PR descriptions |
| 1.5 | Branch protection on `main` (require CI, require PR) | #15 | ~5 min | Prevents direct push to main, enforces PR workflow |

### Agent configuration

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 1.6 | Create `.claude/settings.json` with `permissions.deny` | #17 | ~10 min | Block agent reads of `.env.sensitive` files |
| 1.7 | Fix `/selfimprove` with abstraction guidance | #22 | ~10 min | Output becomes transferable principles, not task-specific facts |
| 1.8 | Create `/review` command | #19 | ~10 min | Agents have a structured review process |

### Knowledge layer (CLAUDE.md improvements)

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 1.9 | Architecture overview in root CLAUDE.md | #13 | ~30 min | Agents understand the system topology without exploring |
| 1.10 | Definition of Done in root CLAUDE.md | #18 | ~10 min | Agents know when to stop and what "complete" means |
| 1.11 | Escalation protocol in root CLAUDE.md | #16 | ~15 min | Agents know when to proceed vs. stop and ask |
| 1.12 | Generated code boundary rule in root CLAUDE.md | Cross-cutting | ~10 min | Agents never hand-edit generated files |
| 1.13 | Shared resource blast radius in root CLAUDE.md | Cross-cutting | ~10 min | Agents understand jirachi/zoroark/migrations affect multiple modules |
| 1.14 | Create `jirachi/CLAUDE.md` | #13 | ~15 min | Shared library has agent guidance; `/selfimprove` stops failing |
| 1.15 | Create `zoroark/CLAUDE.md` | #13 | ~15 min | Component library has agent guidance; `/selfimprove` stops failing |

### Tier 1 exit criteria
- All items shipped and committed
- `make verify` works from root
- `make generate` works from root
- CI runs on pull requests
- Branch protection enforced on main
- Root CLAUDE.md contains: architecture overview, Definition of Done, escalation protocol, generated code rules, blast radius documentation
- `.claude/settings.json` exists with deny rules
- `/review` and updated `/selfimprove` commands exist
- jirachi and zoroark have CLAUDE.md files
- Score 2-3 agent sessions with the new harness and add to `sessions.js`

---

## Tier 2 — Medium Effort, High Impact

**Goal:** Build the enforcement and measurement infrastructure that makes the harness self-reinforcing.

### Session management

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 2.1 | Session startup protocol in CLAUDE.md | Anthropic article, #18 | ~15 min | Agents follow a defined startup sequence: read progress, baseline verify, then implement. Prevents cold-start waste and catches broken state early. |
| 2.2 | Cross-session progress file | Anthropic article, #18 | ~20 min | Persistent `PROGRESS.md` or similar tracks what's done, what's in flight, what's next. Survives compaction and new sessions. |
| 2.3 | CLAUDE.md table of contents pattern | OpenAI article | ~30 min | Keep root CLAUDE.md concise (~100 lines) as a map pointing to deeper docs. Prevents context bloat as knowledge grows. |

### Structural tests

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 2.4 | Enterprise feature flag sync test | #10 | ~30 min | Go constants and DB rows must match — agents can't add a feature flag in only one place |
| 2.5 | Generated code boundary test | #10 | ~30 min | Fails if generated files are hand-edited (detected via header comments or file patterns) |
| 2.6 | CLAUDE.md reference validation test | #24 | ~30 min | File paths and Makefile targets referenced in CLAUDE.md must exist |

### Observability

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 2.7 | Preserve Go test JSON output in CI | #23 | ~15 min | Per-test timing and pass/fail data captured as artifact |
| 2.8 | Add code coverage to Go test commands | #23 | ~15 min | Coverage percentage tracked per CI run |
| 2.9 | Add Playwright JUnit XML reporter | #23 | ~10 min | E2E test results captured in machine-readable format |

### Tooling

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 2.10 | `make setup` bootstrap command | #2, #17, #20, multiple articles | ~1-2 hours | Agents can self-provision from a fresh clone. Multiple articles flag "wasted tokens on env setup" as a top failure mode. |
| 2.11 | `make test-unit-single TEST=<name>` | #20 | ~15 min | Agents can re-run individual failing tests |

### Scheduled automation

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 2.12 | Weekly cleanup scan workflow | #24 | ~30 min | Catches entropy between pushes: deadcode, govulncheck, npm audit |

### Tier 2 exit criteria
- Session startup protocol defined and followed
- Progress file pattern established
- Root CLAUDE.md refactored to table-of-contents pattern
- Structural tests run as part of `make verify`
- CI uploads test output JSON and coverage as artifacts
- `make setup` works from a fresh clone
- Weekly scan workflow exists and runs
- Score 3-5 more agent sessions and update dashboard
- Dashboard shows trend lines — are scores improving?

---

## Tier 3 — Medium Effort, Medium Impact

**Goal:** Polish the harness with secondary improvements. Prioritize based on what the data tells us.

| # | Item | Source | Effort | What it enables |
|---|---|---|---|---|
| 3.1 | `/retro` command + improvement proposals file | #22 | ~20 min | Lightweight end-of-session reflection with persistent memory |
| 3.2 | `make stop` service management | #20 | ~30 min | Agents can restart crashed services reliably |
| 3.3 | Frontend dead code detection (knip) | #24 | ~1 hour | Dead TypeScript/Svelte exports caught automatically |
| 3.4 | Orphaned SQL query structural test | #24 | ~1 hour | SQLC queries not called by any handler are flagged |
| 3.5 | Seed data sync validation test | #10 | ~1 hour | Three seed files must agree on key invariants |
| 3.6 | Configuration drift detection test | #24 | ~30 min | Golangci-lint and ESLint configs must match across modules |
| 3.7 | Agent role definitions in CLAUDE.md | #21 | ~20 min | Different workflow lenses documented for coding, review, testing |
| 3.8 | `feature-list.json` | #14 | ~30 min | Machine-readable inventory of features and their code paths. Anthropic article found JSON format critical — agents are less likely to corrupt JSON than Markdown. Consider promoting if session data shows agents struggling with cross-session context. |
| 3.9 | Version pinning (`.tool-versions`) | #17 | ~15 min | Reproducible tool versions across environments |
| 3.10 | Educational linter error messages | #9, OpenAI article, Substack article | ~1-2 hours | Linter failures double as instructional context for the agent's next attempt. Phase 1: document common fixes in CLAUDE.md. Phase 2: custom lint rules with teaching messages. |
| 3.11 | "Build to delete" modularity principle | Phil Schmid article | ~15 min | Document in CLAUDE.md: harness infrastructure should be modular enough to replace as models improve. Avoid over-engineering control flow — each new model release renders yesterday's logic obsolete. |

### Tier 3 exit criteria
- Items shipped based on priority determined by metric data
- Dashboard shows sustained improvement trend
- Can answer: "at what difficulty level do agents start to struggle, and has that threshold moved?"

---

## Deferred (Not Now)

These were proposed in audits but are explicitly deferred — either because they depend on infrastructure not yet built, or because they're not justified for a solo developer:

| Item | Source | Why deferred |
|---|---|---|
| CODEOWNERS | #19 | Only meaningful with multiple reviewers |
| Multi-agent signaling/communication | #21 | Only needed when running parallel agents |
| Agent sandbox (Docker/Codespaces) | #17 | CLI works for solo developer |
| Quality grading agent | #21, #24 | Needs metrics infrastructure first |
| Devcontainer.json | #17 | Nice-to-have, not blocking anything |
| ADR directory | #13 | Good practice, but not high-leverage now |
| SQLC Querier mock layer | #8 | High value but high effort — revisit after quick wins prove the approach |
| Periodic metrics reporting script | #23 | Dashboard suffices for now |
| Pre-commit hooks | #9 | CI gates provide the same enforcement |

---

## How to Use This Plan

1. **Work through tiers in order** — Tier 0 → 1 → 2 → 3
2. **Score every agent session** — add to `sessions.js`, check the dashboard
3. **Check the data before moving to the next tier** — are scores improving? If not, investigate why before adding more improvements
4. **Adjust** — if a Tier 3 item turns out to be high-impact based on session data, promote it. If a Tier 1 item doesn't help, understand why.
5. **Update this plan** — mark items as done, add new items discovered during implementation

The goal is not to complete every item. The goal is to raise the difficulty threshold at which agents start to struggle — and to have the data to prove it.
