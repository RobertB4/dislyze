# Workstream #18 — Agent Workflow & Orchestration

## Current Setup

There is no formal agent workflow system. The current model is entirely human-directed:

```
Human gives instruction → Agent works → Human evaluates result
```

All task selection, scoping, progress tracking, and completion criteria live in the human's head and the active conversation window. When the session ends, all context is lost except what was committed to git or written into documentation.

### What exists

The 6 custom commands in `.claude/commands/` are the only structured workflow:

| Phase | Command | What it enforces |
|---|---|---|
| Planning | `/plan` | Research first, multiple solutions, pros/cons, never assume |
| Implementation | `/implement` | Re-read plan, surface uncertainties, then code |
| Integration testing | `/integration <endpoint>` | Read implementation + helpers + existing tests → propose → wait |
| E2E testing | `/e2e <feature>` | Same pattern as integration, focused on security and edge cases |
| Self-improvement | `/selfimprove` | Analyze conversation, propose CLAUDE.md improvements |
| Tone | `/honest` | Enable direct feedback mode |

These commands create structured phases but don't address the broader lifecycle: what to work on, how to track progress, when work is done, or how context survives across sessions.

## What's Already Agent-Friendly

### 1. Custom commands encode a planning → implementation → testing pipeline

The `/plan` → `/implement` → `/integration` or `/e2e` sequence is a genuine workflow pipeline. Each command forces research-before-action, creates explicit pause points for human review, and scopes the agent's behavior. This is the strongest workflow artifact in the repository.

### 2. `/plan` creates a natural task intake step

By forcing the agent to research, propose multiple solutions, and explicitly ask clarifying questions before coding, `/plan` creates a structured intake process. The output — multiple options with tradeoffs — is a decision artifact that persists in the conversation.

### 3. HARNESS.md demonstrates stateful task tracking

The workstream audit table in HARNESS.md tracks workstream status across sessions. An agent starting a new session can read HARNESS.md and understand what's been done and what's pending. This pattern — a persistent, machine-readable status document — is exactly what product task tracking needs.

### 4. Test commands enforce "wait for instructions" pattern

`/e2e` and `/integration` both end with "wait for instructions from me before you do anything else." This creates a hard gate between the agent's analysis and its execution — the human must explicitly authorize the next step. This is the strictest workflow gate in the repository.

## What's NOT Agent-Friendly

### 1. No task source of truth (High Impact)

An agent starting a new session cannot answer "what should I work on?" without asking the human. There is no machine-readable backlog, no issue tracker, no task file, no priority list. The only forward-looking enumeration in the repository is the HARNESS.md workstream table — which tracks harness meta-work, not product features.

This means every session begins with a cold start. The human must re-explain the context, the goal, and the scope — or the agent must ask.

**Improvement strategy:** This is addressed at two levels:

1. **`feature-list.json`** (from [#14 Feature tracking](14-feature-tracking.md)) — gives agents a map of what exists. Not a task list, but a feature inventory that provides context for any task.
2. **GitHub Issues as task source** — when agents need to work from a queue, GitHub Issues with structured templates provide: description, acceptance criteria, priority, and status. Combined with GitHub MCP or `gh` CLI access, agents can read their assigned issues.

For a solo developer, the human verbal assignment works. When scaling to multiple agents, a machine-readable task queue becomes essential.

### 2. No completion criteria defined (High Impact)

"Done" is whatever the human accepts in conversation. No document defines what complete means for any type of work. An agent could say "I'm done" having:

- Written code but not run tests
- Run tests but not checked CI
- Made changes but not committed
- Committed but not opened a PR

Without a definition of done, agents either over-deliver (running every possible check) or under-deliver (stopping after writing code).

**Improvement strategy:** Add a "Definition of Done" section to the root CLAUDE.md. The principle: **done means the work is verifiable, not just written.**

```markdown
## Definition of Done

Before reporting work as complete:
1. Code compiles / builds without errors
2. Existing tests still pass (run the relevant test suite)
3. New tests written for new behavior
4. Changes committed to a feature branch
5. PR opened with a description of what changed and why
```

This is abstract enough to apply to any task type while concrete enough to be actionable. The specific verification steps (which test suite, which CI checks) depend on the sub-project CLAUDE.md files.

### 3. No cross-session context persistence (High Impact)

Every session starts from zero. The only things that survive across sessions:

| Survives | What it preserves |
|---|---|
| Git history | Code changes (but minimal "why" — one-line commits, no PR descriptions) |
| CLAUDE.md files | Conventions and patterns (but not task state) |
| HARNESS.md | Workstream status (coarse-grained, harness-only) |

What does NOT survive: what the agent was working on, what decisions were made, what partial work exists, what questions remain open, what the next step was.

An agent resuming work after a session break must either explore the codebase from scratch or rely on the human to re-explain the context. This is expensive (time, tokens) and error-prone (human might forget details).

**Improvement strategy:** Define a session handoff convention. The principle: **the end of a session should produce an artifact that the next session can consume.**

Options:
1. **PR descriptions as session state** — if every task lives on a branch with a PR, the PR description captures what was done, what's pending, and what decisions were made. The next session reads the PR.
2. **Commit messages with context** — conventional commits with bodies that explain the "why" give future sessions breadcrumbs through git history.
3. **Session summary in conversation** — at session end, the agent writes a structured summary (task, decisions, progress, next steps). This persists in Claude Code's conversation history.

The PR-based approach is the most natural — it ties context to the work artifact itself and doesn't require any custom tooling.

### 4. No scope definition at task intake (Medium Impact)

When a human gives an instruction, there's no framework for determining: is this one task or many? What's in scope? What's explicitly out of scope? What are the acceptance criteria?

The root CLAUDE.md says "focus exclusively on the scope of the task at hand" — but scope is never formally defined. The `/plan` command forces analysis of implementation options, but doesn't force boundary definition.

Without explicit scope, agents either interpret too broadly (making unrelated changes) or too narrowly (missing obvious adjacent work that should be included).

**Improvement strategy:** Add a scope definition step to the `/plan` command:

```markdown
Before proposing solutions, define the task scope:
- **In scope**: What this task will accomplish
- **Out of scope**: What this task will NOT touch, even if related
- **Acceptance criteria**: How we know this task is done
```

This scope definition becomes a contract that the agent and human agree on before implementation begins. It also provides the completion criteria for "when am I done?"

### 5. No "done" declaration protocol (Medium Impact)

There is no standard for how an agent reports completion. Currently, the agent says some variant of "I'm done" in conversation. There is no structured format for: what was done, what was tested, what changed, what the human should review.

**Improvement strategy:** This is naturally solved by the PR-based workflow proposed in [#15 Version control](15-version-control-strategy.md). When work is done, the agent:

1. Commits to the feature branch
2. Opens a PR with a structured description (from a PR template)
3. Reports the PR link in conversation

The PR becomes the "done" artifact — it captures the changes, the description, the CI results, and the review surface. The human reviews the PR, not the conversation.

### 6. No task decomposition guidance (Medium Impact)

When given a large or ambiguous task, agents have no guidance on how to break it down. Should a "build feature X" task be one PR or five? Should the agent implement backend, frontend, and tests in one session or propose a phased approach?

**Improvement strategy:** Add task decomposition principles to the root CLAUDE.md. The key insight: **one PR should represent one logical change that can be independently reviewed and tested.** If a task requires multiple logical changes, it should be multiple PRs. This keeps each unit of work reviewable, testable, and revertable.

### 7. No agent self-verification step (Low Impact)

No CLAUDE.md instructs agents to verify their work before reporting done. An agent could push code that fails tests, breaks the build, or violates lint rules without any prompt to check first.

The CI pipeline provides external verification, but agents don't proactively run it. They rely on CI to catch problems after the fact — which means slower feedback loops and more fix-up commits.

**Improvement strategy:** This is partially addressed by the "Definition of Done" (#2 above). Additionally, the `/implement` command could include a post-implementation verification step: "After implementing, run the relevant test suite and lint checks before reporting completion."

## The Workflow Lifecycle

The current workflow is phase 2 only (the middle). The full lifecycle an agent-first workflow needs to cover:

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  1. INTAKE         2. PLANNING         3. IMPLEMENTATION            │
│  ─────────         ───────────         ──────────────────           │
│  What to work on   /plan               /implement                   │
│  Scope definition  Research            Write code                   │
│  Acceptance         Multiple options   Commit to branch             │
│  criteria          Human approval                                   │
│                                                                     │
│  4. VERIFICATION   5. DELIVERY         6. HANDOFF                   │
│  ────────────────  ──────────          ────────────                  │
│  Run tests         Open PR             Session summary              │
│  Check lint        Structured          Context for next             │
│  Self-review       description         session                      │
│                    Human review                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

Today, phases 2-3 are partially covered by custom commands. Phases 1, 4, 5, and 6 have no structure.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No task source of truth | High | `feature-list.json` for context, GitHub Issues for task queue |
| No completion criteria defined | High | Add "Definition of Done" to root CLAUDE.md |
| No cross-session context persistence | High | PR descriptions as session state, conventional commits |
| No scope definition at task intake | Medium | Add scope definition step to `/plan` command |
| No "done" declaration protocol | Medium | PR-based workflow solves this naturally |
| No task decomposition guidance | Medium | Add "one PR = one logical change" principle to CLAUDE.md |
| No agent self-verification step | Low | Include verification step in Definition of Done |

Agent workflow today works because a human is always in the loop, compensating for every gap. The human selects the task, defines the scope, tracks progress, knows when it's done, and carries context across sessions. This works for solo interactive development. It does not scale to autonomous or parallel agent execution.

The two structural changes that would most improve the workflow: **a PR-based development cycle** (branch → work → PR → review → merge) and **a Definition of Done in CLAUDE.md** (so agents know when to stop). Together, they give agents a unit of work (the PR), a completion signal (CI passes + human approves), and an artifact that persists context (the PR description).
