# Workstream #19 — Agent Review Pipeline

## Current Setup

There is no review pipeline. Code goes directly from agent (or human) to `main` with no structured review step. The 3 PRs that exist in the repository's history all have empty bodies, zero reviews, zero comments, and were self-merged immediately.

### What exists

| Component | Status |
|---|---|
| PR template | Does not exist |
| CODEOWNERS | Does not exist |
| Review slash command | Does not exist |
| Merge criteria | Not defined anywhere |
| Branch protection | Not configured (or not codified) |
| CI on pull_request trigger | Does not exist — CI only runs on push |
| Pre-commit hooks | Does not exist |
| Code coverage reporting | Does not exist |
| Third-party quality tools | Does not exist |

The only review-adjacent infrastructure: CI workflows (11 total) that run on push, golangci-lint configs per Go project, ESLint/Prettier for frontend, and vulnerability scanning (govulncheck, npm audit, GCP Artifact Registry scan).

## What's Already Agent-Friendly

### 1. CLAUDE.md quality principles provide review criteria

The root CLAUDE.md defines principles that a reviewing agent can evaluate against: follow existing patterns, prefer simplicity, use existing types, locality of behavior, comments explain "why" not "what." These are abstract enough to apply to any PR while specific enough to be actionable.

### 2. CI checks exist and would work on PRs with minimal changes

All 11 CI workflows trigger on push to any branch. Adding `pull_request` triggers is a configuration change — no new infrastructure needed. Once PRs are used, CI results become available as merge gates.

### 3. The `/honest` command establishes a direct feedback culture

While not a review command, `/honest` trains agents to give blunt feedback rather than rubber-stamp approvals. This tone is what a reviewing agent needs — the willingness to reject or request changes.

### 4. Sub-project CLAUDE.md files define domain-specific conventions

`lugia-backend/CLAUDE.md` documents: middleware ordering, validation patterns, error handling, enterprise feature implementation steps, context management, database query optimization. A reviewing agent working in lugia-backend has concrete conventions to check against. The coverage is uneven (lugia-backend is thorough, others are thin), but the pattern is established.

## What's NOT Agent-Friendly

### 1. No review command or structured review process (High Impact)

There is no `/review` command. An agent asked to review a PR has no guidance on:
- What to check (correctness? style? security? performance? scope?)
- What order to check it in (high-risk first? top-down?)
- What the accept/request-changes/reject criteria are
- How to communicate feedback (inline comments? summary? both?)
- Whether to run the code locally or just read the diff

Without a structured review process, a reviewing agent either does too little (reads the diff, says "looks good") or too much (re-reads the entire codebase for context it doesn't need).

**Improvement strategy:** Create a `/review` command in `.claude/commands/review.md` that encodes a structured review process:

```markdown
Review the PR: $ARGUMENTS

Follow this review process:

1. **Understand the scope**: Read the PR description. What was the task? What should have changed?
2. **Read the diff**: Review every changed file. For each change, check:
   - Does it follow existing patterns in the codebase?
   - Is it within the stated scope?
   - Are there security concerns?
   - Are there performance concerns?
3. **Check conventions**: Read the relevant CLAUDE.md files. Does the code follow documented conventions?
4. **Check tests**: Are there tests for new behavior? Do existing tests still pass?
5. **Check for omissions**: Is anything missing that the task required?
6. **Summarize**: Provide a structured review with: what's good, what needs changes, and a clear accept/request-changes verdict.

Be direct. If the code is wrong, say so. If it's good, say so briefly.
```

The key principle: **a review command should produce a structured verdict, not a conversation.**

### 2. No PR template (High Impact)

Without a PR template, the author agent has no guidance on what to include in the PR description. All 3 historical PRs have empty bodies. A reviewing agent receiving a PR with no description has no context for the change — it must reverse-engineer intent from the diff.

**Improvement strategy:** Create `.github/PULL_REQUEST_TEMPLATE.md`:

```markdown
## What changed
<!-- Brief description of the changes -->

## Why
<!-- What problem does this solve? What task/issue does this address? -->

## How to verify
<!-- Steps to test these changes -->

## Scope
<!-- What's in scope and what's explicitly NOT in scope -->
```

The template serves both the author (forces them to articulate what they did) and the reviewer (gives them context before reading the diff). For agents specifically, the "Scope" section is critical — it tells the reviewer what to check and what to ignore.

### 3. No merge criteria defined (High Impact)

There is no definition of what must be true before a PR can be merged. Without merge criteria, the review is subjective — the reviewer has no bar to hold the PR to.

**Improvement strategy:** Define merge criteria in the root CLAUDE.md or in the PR template. The criteria should be objective and verifiable:

```markdown
## Merge Criteria

A PR can be merged when:
- CI passes (all checks green)
- Changes are within the stated scope
- New behavior has tests
- No security concerns identified
- Code follows documented conventions
- PR description explains what changed and why
```

The key insight: **merge criteria should be things a reviewing agent can verify mechanically or by reading code.** Subjective criteria ("is the code elegant?") create inconsistency between reviewers.

### 4. CI doesn't run on pull_request events (High Impact)

All CI workflows trigger on `push` only — no `pull_request` trigger exists. This means when a PR is opened, there are no CI results to review. The reviewing agent (or human) must manually verify that CI would pass, or wait for the author to push and check the branch's push-triggered results.

This was flagged in [#11 CI/CD pipeline](11-cicd-pipeline.md). Adding `pull_request` triggers to CI workflows is the prerequisite for CI-as-merge-gate.

**Improvement strategy:** Add `pull_request` triggers to all CI workflows (tracked in workstream #11).

### 5. No CODEOWNERS file (Medium Impact)

No mechanism exists to automatically route PRs to reviewers. In a multi-agent environment, CODEOWNERS would define which review agent is responsible for which parts of the codebase.

For a solo developer with one agent, CODEOWNERS is unnecessary. But as the project moves toward agent-to-agent review, it becomes the routing mechanism — which agent reviews which domain.

**Improvement strategy:** Create a basic `CODEOWNERS` file. Even with a single reviewer, it documents ownership:

```
# Default
* @robert

# Domain-specific (future: route to specialized review agents)
/lugia-backend/  @robert
/giratina-backend/  @robert
/jirachi/  @robert
/database/  @robert
/lugia-frontend/  @robert
/giratina-frontend/  @robert
/zoroark/  @robert
```

### 6. No self-review step in the workflow (Medium Impact)

The current workflow (plan → implement → test) has no self-review step between implementation and delivery. An agent finishes coding, runs tests, and reports done — without ever re-reading its own changes from a reviewer's perspective.

Self-review catches a class of errors that implementation misses: scope creep, forgotten edge cases, inconsistent naming, changes that are technically correct but don't match the conventions.

**Improvement strategy:** Add a self-review step to the agent workflow. This could be:

1. **Post-implementation in `/implement`** — after coding, the agent reviews its own diff before committing
2. **A separate `/self-review` command** — invoked between implementation and PR creation
3. **Built into the "Definition of Done"** — "before opening a PR, review your own diff"

The lightest approach: add to the Definition of Done (from [#18 Workflow](18-agent-workflow-orchestration.md)): "Review your own changes before committing. Check for scope creep, convention violations, and missing tests."

### 7. No branch protection enforcing reviews (Medium Impact)

Even if a review process exists, nothing technically prevents merging without review. An agent could push to a branch and merge its own PR without any approval.

**Improvement strategy:** Enable branch protection on `main` requiring:
- At least one approval (even self-approval is better than none — it forces the PR to exist)
- CI checks to pass
- Squash merge only

For a solo developer, the value isn't in the approval gate — it's in the CI gate and the PR structure. The approval becomes meaningful when agent-to-agent review is introduced.

### 8. No feedback loop from review to improvement (Low Impact — future)

When a reviewer finds a problem, there's no mechanism to feed that finding back into the system. A common review finding ("agents keep forgetting to add tests for error paths") should eventually become a CLAUDE.md rule or a structural test — but there's no process for that loop.

**Improvement strategy:** This connects to [#22 Agent retrospective / self-improvement loop](22-agent-retrospective.md). For now, the `/selfimprove` command is the closest thing — it analyzes conversation patterns and proposes CLAUDE.md changes. A review-to-improvement loop would extend this: common review findings get proposed as CLAUDE.md rules or structural tests.

## The Review Pipeline Vision

The full pipeline, from PR creation to merge:

```
Agent opens PR
    │
    ├── PR template filled (what, why, how to verify, scope)
    │
    ▼
CI runs (triggered by pull_request event)
    │
    ├── generate check → lint → typecheck → structural → unit → integration → e2e
    │
    ▼
Review agent assigned (via CODEOWNERS or manual)
    │
    ├── /review command: structured review process
    ├── Check scope, conventions, tests, security
    │
    ▼
Verdict: approve / request changes
    │
    ├── If request changes → author agent fixes → re-review
    │
    ▼
Merge criteria met → squash-merge to main
    │
    ▼
Branch auto-deleted
```

Today, none of this infrastructure exists. But the prerequisites are identified across workstreams #11 (CI triggers), #15 (branch protection, PR workflow), #18 (definition of done), and this workstream.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No review command or structured review process | High | Create `/review` command with structured checklist |
| No PR template | High | Create `.github/PULL_REQUEST_TEMPLATE.md` |
| No merge criteria defined | High | Define objective merge criteria in CLAUDE.md |
| CI doesn't run on pull_request events | High | Add pull_request triggers (workstream #11) |
| No CODEOWNERS file | Medium | Create basic CODEOWNERS for routing |
| No self-review step in workflow | Medium | Add self-review to Definition of Done |
| No branch protection enforcing reviews | Medium | Enable branch protection on main |
| No review-to-improvement feedback loop | Low | Connect to workstream #22 retrospective loop |

The review pipeline is the most "future-facing" workstream so far — it matters most when agents work autonomously and review each other's work. For a solo developer interacting with one agent, the human IS the review pipeline. But the infrastructure — PR templates, merge criteria, CI gates, branch protection — has value even in the solo case because it creates the structure that prevents mistakes from reaching main. The review command and agent-to-agent review become important when scaling beyond one agent.
