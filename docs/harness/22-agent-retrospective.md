# Workstream #22 — Agent Retrospective / Self-Improvement Loop

## Current Setup

The self-improvement loop exists in embryonic form. One mechanism — the `/selfimprove` command — is implemented and has been exercised once (commit `73dcb18`, which expanded `lugia-backend/CLAUDE.md` by 88 lines with security patterns and context management guidance). Everything else — automatic triggers, proposal staging, cross-session pattern detection, feedback from reviews — does not exist.

### What exists

| Component | Status |
|---|---|
| `/selfimprove` command | Implemented — analyzes conversation vs. CLAUDE.md, proposes changes, human approves |
| Evidence of use | One commit: `73dcb18` ("self improve claude md") |
| Harness audit docs | 21 audit files in `docs/harness/` — the meta-improvement process for the harness itself |
| HARNESS.md tracking | Persistent cross-session workstream status |

### The current loop

```
Human invokes /selfimprove at end of session
    │
    ▼
Agent analyzes conversation history vs. all CLAUDE.md files
    │
    ▼
Agent proposes changes one at a time
    │
    ▼
Human approves or rejects each proposal inline
    │
    ▼
Agent writes approved changes into CLAUDE.md files
    │
    ▼
Human commits → next session benefits
```

## What's Already Agent-Friendly

### 1. `/selfimprove` has a well-defined process

The command has a clear 4-phase structure: analysis → interaction → implementation → structured output. Each proposal includes: what the issue is, what the change would be, and how it improves agent performance. The human-in-the-loop approval prevents bad changes from landing.

### 2. The loop has been exercised once

Commit `73dcb18` shows the loop produced output: security best practices, middleware ordering, context management patterns, and query optimization — all surfaced from a single agent session and written back into the knowledge layer. However, the output quality was mixed (see finding #2 below).

### 3. Harness audits are themselves a self-improvement artifact

The 21 audit documents in `docs/harness/` are the output of a meta-level improvement process. They identify gaps, propose strategies, and cross-reference each other. This is the retrospective loop operating at the harness level — and it's thorough.

### 4. CLAUDE.md files are the durable improvement target

Improvements land in CLAUDE.md files, which every future session reads. This means improvements compound: a convention added to CLAUDE.md in session N shapes every session after N. The knowledge layer is the right target for persistent improvements.

## What's NOT Agent-Friendly

### 1. No automatic retrospective trigger (High Impact)

The `/selfimprove` command must be manually invoked. The stated goal of workstream #22 — "after every task, agents reflect abstractly" — is not implemented. In practice, retrospectives happen only when the human remembers to run them, which means they happen rarely.

The single evidence of use (one commit) suggests the loop runs infrequently despite being available.

**Improvement strategy:** Make the retrospective a natural part of the workflow rather than a separate step to remember. Two approaches:

1. **Integrate into the "done" step** — add a retrospective prompt to the Definition of Done (from [#18](18-agent-workflow-orchestration.md)): "Before closing the session, briefly reflect: what was harder than expected? What convention was missing? What would have helped?"
2. **Create a `/retro` command** — lighter than `/selfimprove`, focused on capturing observations rather than immediately implementing changes:

```markdown
Reflect on the work you just completed:

1. What was harder than expected? Why?
2. Did you encounter any missing conventions or unclear guidance in CLAUDE.md?
3. Were there patterns you had to figure out by reading code that should be documented?
4. What would have made this task easier for an agent?

Document your observations as a structured list. Do NOT implement changes — just capture the observations for later review.
```

The key difference: `/selfimprove` analyzes and implements in one step. `/retro` captures observations that accumulate for batch review. Both have value; `/retro` is more likely to be run because it's faster and doesn't require the human to approve each change inline.

### 2. `/selfimprove` output tends toward concrete facts, not abstract principles (High Impact)

In practice, the one time `/selfimprove` was used, the output skewed toward concrete, task-specific facts — things the agent learned about the specific feature it just implemented — rather than abstract principles that would help future agents navigate similar situations. This is the most fundamental quality problem with the current loop.

The distinction:

| Concrete (low transfer value) | Abstract (high transfer value) |
|---|---|
| "The IP whitelist table has columns X, Y, Z" | "When adding enterprise features, always follow the 4-file pattern documented in CLAUDE.md" |
| "The auth middleware is called LoadTenantAndUserContext" | "Middleware ordering matters — new middleware should be added relative to existing middleware based on what context it needs and what context it provides" |
| "The SAML config is stored in tenant settings" | "When integrating third-party auth protocols, the configuration should live in tenant-scoped settings, not in environment variables, because it varies per tenant" |

Agents naturally produce concrete output because they just finished a concrete task. Abstracting from "what I did" to "what principle should guide future agents" requires explicit prompting.

**Improvement strategy:** The `/selfimprove` command needs stronger guidance on abstraction level. Add instructions like:

```markdown
IMPORTANT: Your improvements must be ABSTRACT, not specific to the task you just completed.

- BAD: "Document that the IP whitelist middleware runs after LoadTenantAndUserContext"
  (This is a concrete fact about one feature. It helps no one working on other features.)
- GOOD: "When adding new middleware, document its position relative to existing middleware and what context it requires from prior middleware"
  (This is an abstract principle that helps any agent adding any middleware.)

Ask yourself: "Would this improvement help an agent working on a completely different feature?" If not, make it more abstract.
```

This is the same principle that applies to the harness audits themselves: improvements should teach the mental model, not the specific facts. An agent that understands the principle can derive the correct behavior for any scenario.

### 3. No proposal staging or backlog (High Impact)

When `/selfimprove` runs, proposals surface in conversation. Rejected proposals disappear. Deferred proposals disappear. Only approved-and-applied changes survive. There is no record of what was proposed and not accepted, what was deferred for later, or what patterns keep appearing across sessions.

This means the improvement loop has no memory. A proposal rejected in session 5 might be re-proposed in session 15 with no awareness that it was already considered. A pattern that appears in 3 sessions but is individually too small to act on is never aggregated.

**Improvement strategy:** Create an improvement proposals file — a lightweight staging area:

```markdown
# Improvement Proposals

## Pending
- [2025-06-20] lugia-backend: Document the IP whitelist middleware ordering constraint → Approved, applied
- [2025-07-15] root: Add architecture overview to CLAUDE.md → Deferred, tracked in workstream #13

## Rejected (with reason)
- [2025-06-20] Add $effect lint rule → Rejected: too aggressive, added as CLAUDE.md guidance instead
```

This doesn't need to be complex. A markdown file with date, scope, proposal, and status is sufficient. The value is in persistence — proposals survive across sessions and can be reviewed in aggregate.

### 4. Scope limited to CLAUDE.md files (Medium Impact)

`/selfimprove` can only modify CLAUDE.md files. But the harness includes: custom slash commands, Makefile targets, structural tests, CI workflows, lint rules, and documentation. An agent that realizes "there should be a `/review` command" or "the root Makefile needs a `verify` target" has no mechanism to propose those changes through the self-improvement loop.

**Improvement strategy:** Expand the scope of what the retrospective can propose. The `/retro` command (proposed in #1 above) naturally handles this — it captures observations, not implementations. An observation like "I needed a root-level make verify command but it didn't exist" can be acted on regardless of which file needs to change.

For `/selfimprove` specifically, the command could be extended to propose changes to slash commands and docs — not just CLAUDE.md files. But the simpler path is: use `/retro` for broad observations, use `/selfimprove` for CLAUDE.md-specific improvements.

### 5. Single-session analysis only (Medium Impact)

`/selfimprove` reads the current conversation history. It cannot synthesize patterns across multiple sessions. An improvement that emerges from the pattern of 5 sessions — "agents consistently struggle with the auth middleware ordering" — is invisible to a single-session analysis.

**Improvement strategy:** Cross-session synthesis requires persistent observations. If `/retro` captures observations into a file (proposal #1 and #2 above), then `/selfimprove` can be extended to read that file in addition to the conversation history. The observations file becomes the agent's "memory" of past struggles.

### 6. No feedback loop from reviews to improvements (Medium Impact)

When a reviewer (human or agent) finds a recurring problem in PRs, there's no path from that finding to a CLAUDE.md rule or structural test. The review happens, the feedback is given, but the systemic fix is never proposed.

This was flagged in [#19 Agent review pipeline](19-agent-review-pipeline.md). The improvement loop is the mechanism that should close this gap — common review findings become improvement proposals.

**Improvement strategy:** Add a step to the review process (from [#19](19-agent-review-pipeline.md)): "After reviewing, if you noticed a pattern that could be prevented by better CLAUDE.md guidance, add it to the improvement proposals file." This turns every review into a potential improvement trigger.

### 7. No quality metrics to measure improvement effectiveness (Low Impact)

There is no measurement of whether improvements actually help. Did adding the security best practices to `lugia-backend/CLAUDE.md` reduce security-related review findings? Did documenting the middleware ordering prevent ordering bugs? Without metrics, the improvement loop runs on intuition.

This is tracked in [#23 Observability & metrics](23-observability-metrics.md). The key metrics: how often agents struggle with a topic before and after an improvement, how many review rounds PRs require, whether structural test failures decrease over time.

**Improvement strategy:** Defer to workstream #23. Metrics are valuable but require the workflow infrastructure (PRs, reviews, CI gates) to generate data. Build the workflow first, measure second.

## The Improvement Loop Vision

The full loop, from struggle to systemic fix:

```
Agent completes a task
    │
    ├── /retro: captures what was hard, what was missing, what would help
    │
    ▼
Observations accumulate in improvement-proposals.md
    │
    ├── Patterns emerge across sessions
    │
    ▼
Human (or agent) reviews proposals periodically
    │
    ├── /selfimprove: analyzes proposals + conversation, drafts CLAUDE.md changes
    │
    ▼
Approved changes land in CLAUDE.md, commands, structural tests, or docs
    │
    ▼
Future sessions benefit from improved harness
    │
    ├── Metrics show whether the improvement helped (workstream #23)
    │
    ▼
Loop continues
```

Today, only the `/selfimprove` → CLAUDE.md path exists. The rest is proposed.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No automatic retrospective trigger | High | Integrate reflection into Definition of Done, create `/retro` command |
| `/selfimprove` output is too concrete, not abstract enough | High | Add explicit abstraction guidance to command — improvements must teach principles, not facts |
| No proposal staging or backlog | High | Create improvement proposals file for persistent observation tracking |
| Scope limited to CLAUDE.md files | Medium | `/retro` captures broad observations; `/selfimprove` handles CLAUDE.md |
| Single-session analysis only | Medium | Persistent observations file enables cross-session pattern detection |
| No review-to-improvement feedback loop | Medium | Add improvement proposal step to review process |
| No quality metrics | Low | Defer to workstream #23 |

The self-improvement loop is the mechanism that makes the harness get better over time. Without it, conventions stale, gaps persist, and agents keep struggling with the same problems. The existing `/selfimprove` command has been exercised once but the output quality was mixed — too concrete, not abstract enough. The most important fix is ensuring improvements teach transferable principles ("how to think about middleware ordering") rather than task-specific facts ("the IP whitelist middleware goes after LoadTenantAndUserContext"). Beyond that, making the loop systematic (automatic triggers, persistent proposals, cross-session patterns) rather than ad-hoc is the structural gap.
