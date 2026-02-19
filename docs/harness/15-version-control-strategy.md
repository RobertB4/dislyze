# Workstream #15 — Version Control Strategy

## Current Setup

Git is used with a single `main` branch. Most work is pushed directly to `main` with no feature branches or PRs. Only 3 PRs have ever been created in the repository's history. There are no commit message conventions, no branch naming rules, no branch protection, no git hooks, and no CLAUDE.md guidance on git workflow.

### Branch structure

- `main` — default branch, receives all direct pushes
- `staging` — long-lived environment branch (relationship to main undocumented)
- 2 stale merged feature branches (`add-pulumi-infra`, `shared-library`)

### Commit style

Inconsistent. ~30% use conventional commit prefixes (`fix:`, `chore:`, `refactor:`), the rest are free-form lowercase imperative sentences. All commits are one-liners — no bodies, no issue references. Typos are present (`giratain`, `vulnarabilities`). Multiple CI-fix commits (`fix ci partially`, `try to fix ci`, `add comment to trigger ci`) are not squashed.

## What's Already Agent-Friendly

### 1. Single branch simplicity

For a solo developer pre-production, pushing directly to `main` is the simplest possible workflow. There's no branch confusion, no stale branch management, no merge conflict complexity. An agent operating in this environment has one branch to work with.

### 2. Lowercase imperative commit style is natural

Where followed, the commit style (`add sso endpoints`, `fix giratina ci`) matches how agents naturally generate commit messages. It's readable and grep-friendly.

### 3. CI runs on all branches

All CI workflows trigger on push to any branch. If feature branches are adopted, CI will work immediately without workflow changes.

## What's NOT Agent-Friendly

### 1. No git workflow documented in CLAUDE.md (High Impact)

Zero CLAUDE.md files mention git, branching, commits, or PRs. An agent has no guidance on:
- Should it create a branch or push to main?
- What should commit messages look like?
- Should it open a PR?
- What merge strategy to use?

Without this, agents will default to whatever behavior they're trained on — which varies between models and may not match the project's intent.

**Improvement strategy:** Add a "Git Workflow" section to the root CLAUDE.md. For a solo developer with agents, the simplest effective workflow is:

```markdown
## Git Workflow

- Create a feature branch for every task: `{type}/{short-description}` (e.g., `feat/add-analytics-page`, `fix/login-redirect-bug`)
- Commit messages use conventional commits: `type: description` (e.g., `feat: add analytics page`, `fix: handle login redirect for expired tokens`)
- Types: feat, fix, refactor, chore, test, docs
- Open a PR against main when the work is complete
- Squash-merge PRs to keep main's history clean
- Never push directly to main
```

### 2. No branch protection on main (High Impact)

Anyone (or any agent) can push directly to `main` with no review, no CI check, and no approval. In an agent-first environment, this means a buggy agent could push broken code to the main branch with no safety net.

This was flagged in [#11 CI/CD pipeline](11-cicd-pipeline.md#7-no-branch-protection-visible-in-the-repository-medium-impact). The fix is the same: configure branch protection requiring CI checks to pass and PR-based merges.

**Improvement strategy:** Enable branch protection on `main`:
- Require PR for all changes (no direct push)
- Require CI checks to pass before merge
- Require squash merge (keeps history clean)

For a solo developer, self-approval is fine — the value is in the CI gate and PR structure, not the human review.

### 3. No commit message convention enforced (Medium Impact)

The current history mixes conventional commits with free-form messages. An agent writing commits will produce whatever style it defaults to — likely different from the last human commit. Over time, the history becomes ungreppable.

**Improvement strategy:** Two levels of enforcement:
1. **Documentation** — Define the convention in CLAUDE.md (see #1 above). This is sufficient for agent behavior since agents read CLAUDE.md.
2. **Tooling** — Add `commitlint` with a `commit-msg` git hook (via Husky or simple git hook) that rejects non-conforming messages. This catches mistakes from both humans and agents. A `commitlint.config.js` with `@commitlint/config-conventional` is ~5 lines of config.

For agents specifically, documentation is more important than tooling — agents follow CLAUDE.md instructions more reliably than humans.

### 4. PRs are not used as a workflow (Medium Impact)

Only 3 PRs exist in the entire history. The current workflow is: write code → push to main → CI runs → done. There's no structured point where changes are reviewed, described, or linked to features.

When agents start working autonomously (workstreams 17-21), PRs become essential — they're the unit of work an agent produces, and the review surface where another agent or human evaluates the change.

**Improvement strategy:** Adopt PRs as the standard workflow. Combined with branch protection (no direct push to main), this naturally enforces: branch → work → PR → CI → merge. Add a PR template (see [#14 Feature tracking](14-feature-tracking.md)) to structure descriptions.

### 5. No automated dependency updates (Medium Impact)

No Dependabot or Renovate configuration exists. Go modules and npm packages will drift without manual intervention. An agent won't know when dependencies have security patches available.

**Improvement strategy:** Add `dependabot.yml` with weekly update checks for Go modules and npm packages:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/lugia-backend"
    schedule:
      interval: "weekly"
  - package-ecosystem: "gomod"
    directory: "/giratina-backend"
    schedule:
      interval: "weekly"
  - package-ecosystem: "gomod"
    directory: "/jirachi"
    schedule:
      interval: "weekly"
  - package-ecosystem: "npm"
    directory: "/lugia-frontend"
    schedule:
      interval: "weekly"
  - package-ecosystem: "npm"
    directory: "/giratina-frontend"
    schedule:
      interval: "weekly"
  - package-ecosystem: "npm"
    directory: "/zoroark"
    schedule:
      interval: "weekly"
```

### 6. CI-fix commits pollute main's history (Low Impact)

Multiple sequential commits like `try to fix ci`, `fix ci partially`, `add comment to trigger ci` are part of main's permanent history. These make it harder for an agent (or human) to understand what changed and why when reading `git log`.

**Improvement strategy:** Squash-merge PRs eliminates this naturally — CI-fix commits stay on the feature branch and are squashed into a single clean commit on main. This is solved by adopting the PR workflow (#4 above).

### 7. Stale merged branches not cleaned up (Low Impact)

`add-pulumi-infra` and `shared-library` were merged but never deleted from the remote. An agent running `git branch -r` sees branches that are no longer active.

**Improvement strategy:** Delete merged branches. Configure GitHub to auto-delete branches after PR merge (repository settings → "Automatically delete head branches").

### 8. `staging` branch relationship undocumented (Low Impact)

A `staging` branch exists but its relationship to `main` is not documented anywhere. Is it a deployment target? A pre-production integration branch? An agent encountering this branch has no context.

**Improvement strategy:** Either document the staging branch's purpose in CLAUDE.md or remove it if it's no longer needed. If staging is a deployment target, document the flow: `feature branch → PR → main → deploy to staging → promote to production`.

## Recommended Git Workflow for Agent-First Development

```
Agent receives task
    │
    ▼
Create branch: feat/short-description
    │
    ▼
Implement (commits: feat: ..., fix: ..., etc.)
    │
    ▼
Push branch, open PR
    │
    ▼
CI runs (generate check → lint → typecheck → structural → unit → integration → e2e)
    │
    ▼
All green → squash-merge to main
    │
    ▼
Branch auto-deleted
```

This workflow gives agents clear instructions, produces a clean history, and creates review surfaces at every step.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No git workflow in CLAUDE.md | High | Add git workflow section with branch/commit conventions |
| No branch protection on main | High | Require PRs + CI checks before merge |
| No commit message convention enforced | Medium | Document in CLAUDE.md, optionally add commitlint |
| PRs not used as workflow | Medium | Adopt PR-based workflow with squash merge |
| No automated dependency updates | Medium | Add Dependabot configuration |
| CI-fix commits in main history | Low | Solved by squash-merge PRs |
| Stale merged branches | Low | Delete and enable auto-delete |
| Staging branch undocumented | Low | Document or remove |

The version control strategy is currently "push to main and hope for the best." For a solo developer pre-production, this has worked fine. But for agent-first development, git workflow becomes infrastructure: agents need explicit rules for branching, committing, and merging, and the CI pipeline needs PR triggers and branch protection to provide the feedback loop. The transition from "push to main" to "branch → PR → CI → merge" is the single most important structural change for enabling autonomous agent work.
