# Workstream #16 — Human Escalation Protocol

## Current Setup

There is no formal escalation protocol. No CLAUDE.md file defines when agents must stop and ask a human. No `.claude/settings.json` restricts what agents can do. The only escalation-like behavior exists in a few soft spots:

- Root CLAUDE.md: "Proactively ask clarifying questions and communicate unknowns/risks before writing code"
- `/plan` command: "NEVER ASSUME OR GUESS — always confirm if you are unsure"
- `/e2e` and `/integration` commands: "Wait for instructions from me before you do anything else"
- `/implement` command: "Is there anything you don't understand or are unsure of?"

These are opt-in (only active when the command is invoked) and behavioral (rely on the agent following instructions, not on mechanical enforcement).

## What's Already Agent-Friendly

### 1. Deploy workflows require manual trigger

All three deploy workflows have `push: branches: none` — deploys can only be triggered via `workflow_dispatch` in the GitHub UI. No agent working on code can accidentally deploy to production. This is the strongest structural escalation boundary in the repository.

### 2. Custom commands encode pause-before-acting patterns

`/plan`, `/implement`, `/e2e`, and `/integration` all create moments where the agent must stop and surface uncertainty or wait for explicit authorization. The `/e2e` and `/integration` commands are the most concrete — they explicitly say "wait for instructions before you do anything else."

### 3. "Accuracy over speed" philosophy creates caution

The root CLAUDE.md's emphasis on understanding before acting, asking clarifying questions, and staying within task scope naturally reduces the risk of agents making autonomous bad decisions. An agent following these principles will be cautious by default.

### 4. Security anti-patterns are documented

`lugia-backend/CLAUDE.md` lists specific things to never do (header-based auth, string feature flags, direct DB calls for security checks). While not framed as escalation triggers, they provide guardrails that prevent specific classes of mistakes.

## What's NOT Agent-Friendly

### 1. No explicit "stop and ask" conditions defined (High Impact)

There is no list of situations where an agent must stop working and escalate to a human. An agent encountering any of these situations has no guidance:

- "I need to modify the database schema"
- "I'm about to delete files or data"
- "I found a security vulnerability in existing code"
- "The task requires changes to auth or middleware"
- "I'm not sure if this is the right approach"
- "The tests I wrote are failing and I don't understand why"
- "I need to modify infrastructure or CI configuration"

Without explicit stop conditions, agents will either stop too often (asking about everything) or not often enough (making autonomous decisions that should have been reviewed).

**Improvement strategy:** Add a "When to Stop and Ask" section to the root CLAUDE.md:

```markdown
## When to Stop and Ask

Always stop and ask a human before:
- Modifying database schema (migrations)
- Changing auth or security middleware
- Modifying CI/CD workflows or infrastructure
- Deleting files, branches, or data
- Making changes outside the scope of your assigned task
- Choosing between multiple valid approaches with significant trade-offs

Stop and surface uncertainty when:
- You cannot find an existing pattern to follow
- Tests are failing for reasons you don't understand
- You discover what appears to be a bug in existing code
- The task requirements are ambiguous
```

### 2. No decision authority boundaries defined (High Impact)

There is no definition of what agents CAN decide autonomously vs. what requires human approval. This matters because the line is different for different types of decisions:

| Decision type | Should be autonomous? |
|---|---|
| Implementation details within a well-defined pattern | Yes |
| Which existing pattern to follow | Yes |
| Variable/function naming within conventions | Yes |
| Creating a new feature domain directory | No — ask first |
| Adding a new enterprise feature | No — requires 4-file change, ask first |
| Choosing between fundamentally different approaches | No — present options |
| Modifying shared library (jirachi/zoroark) | No — blast radius is high |
| Adding new dependencies | No — ask first |

Without these boundaries, agents oscillate between over-autonomy and under-autonomy. The goal is to define a clear line so agents know when to proceed and when to pause.

**Improvement strategy:** Add a "Decision Authority" section to the root CLAUDE.md that categorizes decisions into "proceed autonomously" vs. "present options and wait" vs. "stop and ask." Keep it abstract — teach the principle (blast radius, reversibility, convention-following) rather than listing every scenario.

### 3. No mechanical enforcement of escalation (Medium Impact)

All current escalation guidance relies on agents reading and following CLAUDE.md. There are no mechanical stops — no `.claude/settings.json` with `permissions.deny`, no `PreToolUse` hooks, no required approvals.

This matters because even well-intentioned agents can miss CLAUDE.md instructions, especially in long sessions or when context windows are full.

**Improvement strategy:** Add mechanical enforcement for the highest-risk operations:

1. **`permissions.deny` in `.claude/settings.json`** — Block reads of `.env.sensitive` (see [#12 Security boundaries](12-security-boundaries.md))
2. **`PreToolUse` hooks** — Add hooks for high-risk tools:
   - Before `Bash` commands that match patterns like `rm -rf`, `DROP`, `git push --force`, `make deploy`: require confirmation
   - Before `Write` or `Edit` on files matching `database/migrations/*`, `.github/workflows/*`, `infrastructure/*`: require confirmation
3. **Branch protection** (see [#15 Version control](15-version-control-strategy.md)) — Prevents pushing directly to main, creating a natural PR-based review gate

### 4. No protocol for unexpected discoveries (Medium Impact)

An agent working on a task might discover: a security vulnerability, a live bug (like the stale jirachi queries from [#4](04-database-layer.md)), dead code, or architectural issues. Currently, the root CLAUDE.md says "add comments explaining what we want to change and why" — but this is inadequate for discoveries that need immediate attention.

**Improvement strategy:** Add guidance for unexpected discoveries:

```markdown
## Unexpected Discoveries

If you discover something concerning while working on a task:
1. Document what you found and where
2. Assess severity: is this a security issue, a live bug, or a code quality concern?
3. For security issues or live bugs: stop your current task and report immediately
4. For code quality concerns: add a comment in the code and mention it when you report your task results
5. Never fix an out-of-scope issue without asking first — even if the fix seems obvious
```

### 5. No escalation for irreversible actions (Medium Impact)

Some actions are hard to undo: database migrations (especially destructive ones), git force pushes, file deletions, dependency removals. There is no concept of "this cannot be undone — confirm before proceeding" in any documentation.

The deploy workflows have a mechanical gate (manual trigger), but local destructive actions have no gate at all. An agent could run `database/drop.sql` against the local database or delete files without any guardrail.

**Improvement strategy:** Define a reversibility principle in CLAUDE.md:

```markdown
## Reversibility Principle

Before taking any action, consider: can this be easily undone?
- Reversible (safe to proceed): editing files, creating new files, running tests, creating branches
- Hard to reverse (ask first): database migrations, deleting files, modifying CI/CD, adding dependencies
- Irreversible (never without explicit instruction): dropping databases, force pushing, deleting branches, deploying
```

### 6. No escalation path for agent-to-agent coordination (Low Impact — future)

When multiple agents work on the same codebase (workstreams 18-21), they need escalation paths to each other — not just to humans. For example: "Agent A modifying the database schema should notify Agent B working on the frontend." This is a future concern but worth noting in the protocol design.

**Improvement strategy:** Defer to Phase 2 (agent coordination workstreams). For now, the protocol focuses on agent-to-human escalation. When multi-agent workflows are introduced, extend the protocol with agent-to-agent signaling.

## The Escalation Spectrum

The protocol should define a spectrum, not a binary:

```
Proceed autonomously          Present options          Stop and ask          Never do
─────────────────────────────────────────────────────────────────────────────────────
Follow existing patterns      Multiple valid           Schema changes        Deploy
Variable naming               approaches with          Security/auth         Force push
Test implementation           trade-offs               New dependencies      Drop database
Bug fixes within scope        New feature domain       Infrastructure        Delete branches
Code within assigned task     Shared lib changes       Out-of-scope fixes
```

The principle behind the spectrum: **the more reversible and convention-following an action is, the more autonomous the agent should be. The higher the blast radius and the less precedent exists, the more the agent should escalate.**

## Summary

| Finding | Impact | Action |
|---|---|---|
| No explicit "stop and ask" conditions | High | Add "When to Stop and Ask" to root CLAUDE.md |
| No decision authority boundaries | High | Define autonomous vs. ask-first vs. never categories |
| No mechanical enforcement | Medium | Add `permissions.deny`, `PreToolUse` hooks, branch protection |
| No protocol for unexpected discoveries | Medium | Add discovery handling guidance to CLAUDE.md |
| No escalation for irreversible actions | Medium | Define reversibility principle in CLAUDE.md |
| No agent-to-agent escalation | Low | Defer to Phase 2 multi-agent workstreams |

The escalation protocol is the most "soft" workstream — it's primarily documentation and cultural norms rather than tooling. But it's critical for agent-first development because it defines the boundary between agent autonomy and human oversight. Without it, agents either ask too much (annoying, slow) or too little (dangerous). The goal is a clear, abstract principle — blast radius and reversibility — that agents can apply to any situation, not a list of specific rules they must memorize.
