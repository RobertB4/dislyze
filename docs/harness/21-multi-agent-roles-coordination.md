# Workstream #21 — Multi-Agent Roles & Coordination

## Current Setup

There is one agent: a generalist Claude Code instance running locally via CLI. No agent roles are defined, no coordination mechanisms exist, and the entire workflow assumes a single agent interacting with a single human. The concept of multiple agents working on the same codebase is not represented anywhere in the repository's configuration or documentation.

### Implicit roles from custom commands

The 6 custom commands in `.claude/commands/` create workflow phases that map loosely to agent specializations, but they're used by the same agent instance switching modes — not by distinct agents:

| Command | Implicit Role |
|---|---|
| `/plan` | Analyst / Architect |
| `/implement` | Implementer |
| `/integration` | Integration Test Writer |
| `/e2e` | E2E / Security Test Writer |
| `/selfimprove` | Doc Gardener |
| `/honest` | Critic (tone modifier) |

### Roles named in HARNESS.md but not yet defined

HARNESS.md workstreams #21 and #24 name five agent types: coding, review, doc-gardening, quality-grading, and security. Of these, only doc-gardening has any implementation (`/selfimprove`).

## What's Already Agent-Friendly

### 1. Command-based role switching works for a single agent

The `/plan` → `/implement` → `/e2e` pipeline lets one agent play multiple roles in sequence. This is the right model for a solo developer with one agent — specialization happens through workflow phases, not through separate agent instances.

### 2. `/selfimprove` is a genuine doc-gardening role

The `/selfimprove` command has a defined process (analyze → propose → get feedback → implement), a defined scope (all CLAUDE.md files), and a defined output contract (structured analysis/improvements/final-instructions format). It's the most complete agent role in the repository — a working example of what other roles could look like.

### 3. CLAUDE.md files provide shared knowledge across sessions

Any agent starting a session reads the same CLAUDE.md files and inherits the same conventions. This is the most basic coordination mechanism — shared norms. Even without explicit coordination, two agents following the same CLAUDE.md will produce more consistent code than two agents with no shared context.

### 4. CI workflows provide a universal verification gate

Regardless of which agent produced the code, CI runs the same checks. This is role-agnostic enforcement — it doesn't matter who wrote it, only whether it passes.

## What's NOT Agent-Friendly

### 1. No concept of agent roles in any documentation (High Impact)

No document defines what different agents should do, what their boundaries are, or how they differ from each other. The root CLAUDE.md describes one implicit role: a generalist developer-agent. There is no concept of specialization.

This matters because different types of work require different behaviors. A coding agent should focus on implementation quality. A review agent should focus on convention compliance and scope adherence. A security agent should focus on vulnerability patterns. Without role definitions, every agent defaults to generalist behavior.

**Improvement strategy:** Define agent roles abstractly in the root CLAUDE.md or a dedicated `docs/agent-roles.md`. The goal is not to create rigid personas but to document the different lenses agents should apply depending on their task. The principle: **role defines what to pay attention to, not what to do.**

Roles to define:

| Role | Focus | Commands | When used |
|---|---|---|---|
| Coding Agent | Implementation quality, pattern adherence, test coverage | `/plan`, `/implement` | Feature development, bug fixes |
| Review Agent | Convention compliance, scope adherence, security, completeness | `/review` (to be created) | PR review |
| Test Agent | Coverage, edge cases, security paths, failure modes | `/integration`, `/e2e` | Test writing |
| Doc Gardener | Knowledge accuracy, CLAUDE.md completeness, stale content | `/selfimprove` | After sessions, on schedule |
| Quality Grader | Code quality assessment, technical debt identification | (to be created) | Periodic audit |
| Security Agent | Vulnerability scanning, auth pattern review, input validation | (to be created) | Security-sensitive changes |

### 2. No coordination mechanism between agents (High Impact)

If two agents work on the same codebase, they have no way to:
- Know what the other is working on
- Avoid conflicting changes to the same files
- Signal that a shared resource (database schema, shared library) is being modified
- Hand off work or dependencies

Currently this is not a problem — one agent, one human, sequential work. But the moment parallel agent execution is introduced (workstream #17's sandbox options), coordination becomes critical.

**Improvement strategy:** Coordination should be layered, from simplest to most sophisticated:

**Layer 1 — Branch isolation (no coordination needed)**
Each agent works on its own feature branch. Git handles isolation. Conflicts are resolved at PR merge time. This is the minimum viable coordination model and requires only the branch-per-task convention from [#15 Version control](15-version-control-strategy.md).

**Layer 2 — Task queue prevents overlap**
GitHub Issues with assignment prevent two agents from picking up the same task. An agent checks "is this issue assigned?" before starting. This requires the task queue from [#18 Workflow](18-agent-workflow-orchestration.md).

**Layer 3 — Domain ownership reduces conflicts**
CODEOWNERS-style routing ensures agents specialize by domain. The lugia-backend agent doesn't modify giratina-backend code, reducing the merge conflict surface. This requires the CODEOWNERS file from [#19 Review pipeline](19-agent-review-pipeline.md).

**Layer 4 — Explicit signaling for shared resources (future)**
When an agent modifies a shared resource (database schema, jirachi, zoroark), it signals other agents that may be affected. This could be as simple as a label on the PR or as sophisticated as a message queue. This is a future concern.

Start with Layer 1. It solves most coordination problems without any new infrastructure.

### 3. No `/review` command (Medium Impact)

The reviewer role has no command infrastructure. This was flagged in [#19 Agent review pipeline](19-agent-review-pipeline.md) with a proposed command structure. Without it, review agents have no structured process to follow.

**Improvement strategy:** Create `/review` command (tracked in workstream #19).

### 4. No quality-grading or security agent roles defined (Medium Impact)

HARNESS.md names quality-grading and security agents but neither is defined. These are specialized roles that apply specific lenses:

- **Quality grader**: Assesses technical debt, identifies code that doesn't follow conventions, flags areas that need refactoring. Operates periodically, not on every PR.
- **Security agent**: Reviews auth patterns, input validation, SQL injection vectors, XSS risks, dependency vulnerabilities. Operates on security-sensitive changes.

**Improvement strategy:** Define these as commands (like `/selfimprove` defines doc-gardening):

- **`/quality-audit`** — Read a feature domain's code against CLAUDE.md conventions, identify deviations, propose improvements. Outputs a structured quality report.
- **`/security-review`** — Read changes touching auth, middleware, database queries, or user input. Check against known vulnerability patterns. Outputs a structured security assessment.

These are lower priority than `/review` because they're periodic rather than per-PR.

### 5. No agent-to-agent communication channel (Low Impact — future)

Agents cannot communicate with each other. There is no message queue, no shared state file, no signaling mechanism. In a multi-agent setup, Agent A finishing a database migration has no way to tell Agent B (working on the frontend) that the schema changed.

**Improvement strategy:** Defer until multi-agent execution is actually in use. The simplest channel is git itself — Agent A creates a commit on its branch, Agent B reads the branch. PR descriptions serve as structured messages between author and reviewer agents. For anything more sophisticated, a dedicated coordination mechanism (shared file, GitHub Issues, message queue) would be needed.

### 6. No role-based permissions (Low Impact — future)

All agents have identical access. A doc-gardening agent has the same file system access as a coding agent. A security agent could modify code, not just review it. There's no principle of least privilege.

**Improvement strategy:** When `.claude/settings.json` is created (workstream #17), it could define different permission profiles per role. However, Claude Code currently supports one settings file per project — not per-agent-role. Role-based permissions would require either separate workspaces or a custom launcher that injects role-specific settings. Defer until the tooling supports it.

## The Coordination Spectrum

Like the escalation spectrum from [#16](16-human-escalation-protocol.md), coordination needs exist on a spectrum:

```
No coordination needed          Lightweight coordination          Active coordination
───────────────────────────────────────────────────────────────────────────────────
Independent tasks on             Shared codebase with             Shared resources
separate branches                branch isolation                 (schema, shared libs)

Each agent works alone           Task queue prevents              Agents signal each
and merges via PR                overlap; PR review               other about changes
                                 catches conflicts                to shared resources

Current model                    Near-term target                 Future need
(one agent)                      (parallel agents)                (tightly coupled work)
```

The principle: **coordination overhead should match the coupling between agents' work.** Independent tasks need no coordination. Shared-resource tasks need explicit signaling. Most work falls in the middle — branch isolation plus a task queue.

## Practical Next Steps

For a solo developer with one agent, the immediate value is in defining roles as workflow lenses (not as separate agent instances):

1. **Create `/review` command** — establishes the reviewer role (workstream #19)
2. **Define roles in CLAUDE.md** — so agents know what to focus on depending on their current task
3. **Adopt branch-per-task** — the minimum viable isolation mechanism (workstream #15)
4. **Create CODEOWNERS** — documents domain ownership even before multi-agent (workstream #19)

Multi-agent execution infrastructure (sandboxes, task queues, signaling) is deferred to when parallel agent work actually begins.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No agent roles defined in documentation | High | Define roles as workflow lenses in CLAUDE.md |
| No coordination mechanism between agents | High | Branch isolation (Layer 1), task queue (Layer 2), domain ownership (Layer 3) |
| No `/review` command | Medium | Create review command (workstream #19) |
| No quality-grading or security agent roles | Medium | Define as commands (`/quality-audit`, `/security-review`) |
| No agent-to-agent communication channel | Low | Defer — git and PR descriptions serve as minimal channels |
| No role-based permissions | Low | Defer — Claude Code doesn't support per-role settings |

Multi-agent coordination is the most forward-looking workstream. The current single-agent model works, and most coordination problems are solved by the branch-per-task convention that's already recommended across multiple workstreams. The real value today is in defining roles as different lenses agents apply — not in building coordination infrastructure for agents that don't yet run in parallel.
