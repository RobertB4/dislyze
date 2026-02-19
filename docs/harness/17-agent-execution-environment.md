# Workstream #17 — Agent Execution Environment

## Current Setup

Agents run via Claude Code CLI on the developer's local machine. There is no sandboxing, no remote execution, no agent-specific configuration beyond 6 custom slash commands in `.claude/commands/`.

### What exists

| Component | Status |
|---|---|
| `.claude/commands/` | 6 custom commands: `/plan`, `/implement`, `/e2e`, `/integration`, `/honest`, `/selfimprove` |
| `.claude/settings.json` | Does not exist — no permissions, no hooks, no deny rules |
| `.claude/mcp.json` | Does not exist — no MCP servers configured |
| `devcontainer.json` | Does not exist — no reproducible dev environment spec |
| Git hooks | None — no pre-commit, no commit-msg, no Husky/lefthook |
| Version pinning (local) | None — Go pinned in `go.work` (1.24.12) and CI, Node pinned only in Dockerfiles (v20) |
| Playwright | Configured for E2E tests in Docker (`mcr.microsoft.com/playwright:v1.58.0-jammy`), not directly available to agents |
| Local dev | `make dev` starts 6 services in parallel (both backends, both frontends, sendgrid-mock, keycloak-mock) |

### Execution model

One agent at a time, running locally. The agent shares the developer's full file system, full network access, and full tool access. No isolation, no restrictions, no audit trail beyond conversation history.

## What's Already Agent-Friendly

### 1. Custom slash commands are well-designed workflow gates

The 6 commands in `.claude/commands/` create structured workflow phases:

- **`/plan`** — Forces research-first thinking, multiple solutions with pros/cons, explicit "NEVER ASSUME OR GUESS"
- **`/implement`** — Re-reads the plan before implementing, asks for clarification
- **`/e2e`** and **`/integration`** — Enforce research-first test writing: read the feature, read CLAUDE.md, read setup package, read existing tests, then propose — and "wait for instructions before you do anything else"
- **`/honest`** — Enables direct feedback mode
- **`/selfimprove`** — Structured self-improvement loop across all CLAUDE.md files

These commands are the strongest workflow enforcement mechanism in the repository. They turn what would be vague instructions into repeatable agent processes.

### 2. Local dev environment starts with one command

`make dev` starts all 6 services in parallel. An agent that has a running local environment can iterate quickly — make changes, hit endpoints, run tests. The `Makefile` structure is simple and discoverable.

### 3. Deploy workflows cannot be triggered by agents

All three deploy workflows (`deploy-lugia.yml`, `deploy-giratina.yml`, `deploy-infrastructure.yml`) require `workflow_dispatch` — manual trigger only. No agent working on code can accidentally deploy to production. This is the most important safety boundary in the repository.

### 4. Playwright E2E infrastructure is mature

E2E tests run in Docker Compose with a dedicated Playwright container (`mcr.microsoft.com/playwright:v1.58.0-jammy`). The infrastructure is proven — tests exist, configurations are pinned, the Docker Compose setup handles service orchestration. This is a solid foundation for agent-driven E2E testing.

### 5. Committed `.env` files provide zero-friction local config

All `.env` files contain localhost-only values and are committed to the repository. An agent doesn't need to create, find, or configure environment variables — they're already there. This eliminates a common source of agent confusion.

## What's NOT Agent-Friendly

### 1. No `.claude/settings.json` — no permissions, no hooks, no restrictions (High Impact)

The most important agent configuration file doesn't exist. Without it:

- **No `permissions.deny`** — agents can read any file, including `.env.sensitive` (which exists as a placeholder for production secrets)
- **No `PreToolUse` hooks** — no mechanical enforcement of escalation rules (see [#16 Human escalation protocol](16-human-escalation-protocol.md))
- **No `PostToolUse` hooks** — no audit logging, no post-action validation
- **No allowed/denied command patterns** — agents can run any bash command

This is the highest-leverage single file that could be created. A minimal `settings.json` would:
1. Deny reads of `.env.sensitive` files
2. Add `PreToolUse` hooks for high-risk operations (destructive bash commands, migration file edits, CI workflow edits)

**Improvement strategy:** Create `.claude/settings.json` with:
- `permissions.deny` for `.env.sensitive` reads
- `PreToolUse` hooks for the highest-risk tool patterns identified in [#16](16-human-escalation-protocol.md)

Start minimal — a few critical deny rules and hooks. Expand as patterns emerge from actual agent usage.

### 2. No `make setup` bootstrap command (High Impact)

There is no single command that takes a fresh clone to a working state. An agent (or new developer) must discover the setup steps by reading Makefiles, Dockerfiles, and `.env` files:

1. Install Go 1.24, Node 20, PostgreSQL, goose, sqlc, air, golangci-lint, gotestfmt
2. Start PostgreSQL
3. `make initdb` (run migrations + seed)
4. `make dev` (start all services)

An agent that encounters setup failures has no recovery path documented anywhere.

**Improvement strategy:** Create `make setup` that:
1. Checks prerequisites (Go, Node, PostgreSQL, required tools)
2. Reports what's missing with install instructions
3. Initializes the database
4. Validates the environment is ready

This was also identified in [#2 Development environment](02-dev-environment.md) and [#13 Knowledge architecture](13-knowledge-architecture.md). It's the most-referenced gap across workstreams.

### 3. No version pinning for local development tools (Medium Impact)

Go is pinned in `go.work` (1.24.12) and Node in Dockerfiles (v20), but neither is formally specified for local development. Additional tools have no version pinning at all:

| Tool | Pinned? | Where |
|---|---|---|
| Go | In `go.work` and CI | Not enforced locally |
| Node | In Dockerfiles | Not enforced locally |
| goose | No | Installed ad hoc |
| sqlc | No | v1.28 vs v1.29 skew exists (see [#7](07-generation-pipeline.md)) |
| air | No | Installed ad hoc |
| golangci-lint | No | `@latest` in CI |
| gotestfmt | No | `@latest` in CI |
| Playwright | Docker image pinned | Not relevant for local |

An agent running `go install` or `npm install -g` gets whatever version is current, which may differ from what CI uses or what another agent used yesterday.

**Improvement strategy:** Add a `.tool-versions` file (compatible with `asdf` and `mise`) or equivalent version pinning. At minimum, document expected tool versions in the root CLAUDE.md so agents know what they're targeting.

### 4. `/selfimprove` references missing CLAUDE.md files (Medium Impact)

The `/selfimprove` command references 8 CLAUDE.md files:

```
@CLAUDE.md
@database/CLAUDE.md
@lugia-backend/CLAUDE.md
@giratina-backend/CLAUDE.md
@lugia-frontend/CLAUDE.md
@giratina-frontend/CLAUDE.md
@jirachi/CLAUDE.md          ← does not exist
@zoroark/CLAUDE.md          ← does not exist
```

When an agent runs `/selfimprove`, it will fail to load two of the eight referenced files. This was also flagged in [#13 Knowledge architecture](13-knowledge-architecture.md) — creating CLAUDE.md files for jirachi and zoroark fixes both issues.

**Improvement strategy:** Create `jirachi/CLAUDE.md` and `zoroark/CLAUDE.md` (tracked in workstream #13).

### 5. No MCP server configuration (Medium Impact)

MCP (Model Context Protocol) servers extend agent capabilities with structured tool access. No MCP servers are configured. Potential high-value MCP servers for this project:

- **PostgreSQL MCP** — gives agents structured database access (schema inspection, query execution) without raw `psql` commands
- **GitHub MCP** — structured PR/issue/review access

Note: browser access for agents is handled by [playwright-cli](https://github.com/microsoft/playwright-cli) rather than an MCP server — see [#20 Agent-specific tooling](20-agent-specific-tooling.md). playwright-cli is more token-efficient for agents because it avoids loading large tool schemas and verbose accessibility trees into agent context.

**Improvement strategy:** Evaluate and configure MCP servers in `.claude/mcp.json`. Prioritize:
1. PostgreSQL MCP — gives agents safe, structured database access
2. GitHub MCP — enables PR-based workflows from within agent sessions

### 6. No devcontainer.json for reproducible environments (Medium Impact)

No `.devcontainer/` configuration exists. This matters for two reasons:

1. **GitHub Codespaces** — if the project moves to Codespaces for parallel agent execution, a `devcontainer.json` is required
2. **Reproducibility** — without a devcontainer spec, the "works on my machine" problem applies to agents too. Different local environments produce different results.

**Improvement strategy:** Create a `devcontainer.json` that specifies: base image, required tools and versions, port forwarding, post-create setup commands (`make setup && make initdb`). This is useful regardless of whether Codespaces is adopted — it documents the environment specification.

### 7. Agents cannot visually verify frontend changes during development (Medium Impact)

The hard requirement from the preliminary notes: "agents MUST have access to a browser they can navigate." Currently:

- Playwright is available only inside E2E Docker Compose (not during normal development)
- An agent working on frontend code can only verify changes by reading code — not by seeing them

This gap means agents building UI features operate blind. They can write correct code structurally, but cannot validate visual layout, interaction behavior, or responsive design.

**Improvement strategy:** Install [playwright-cli](https://github.com/microsoft/playwright-cli) (`npm install -g @playwright/cli@latest`). This is a CLI tool from Microsoft built specifically for AI coding agents — it provides browser navigation, screenshots, clicking, typing, and page snapshots through CLI commands. It is more token-efficient than an MCP-based approach because it avoids loading large tool schemas into agent context. Run `playwright-cli install --skills` to install agent-readable skill files for Claude Code. See [#20 Agent-specific tooling](20-agent-specific-tooling.md) for details.

### 8. No agent sandbox for parallel/autonomous work (Low Impact — future)

The preliminary notes document four options: Claude Code CLI (current), Claude Code Remote (blocked by browser proxy issue), Docker sandboxes, and GitHub Codespaces. For a solo developer, local CLI execution works. But it blocks the developer's machine and limits to one agent at a time.

**Improvement strategy:** Defer full sandbox implementation. When parallel agent execution becomes a priority, the decision tree is:

1. If Claude Code Remote resolves the browser limitation → use it (simplest)
2. If reproducible isolation is needed → Docker sandboxes with custom image
3. If managed infrastructure is preferred → GitHub Codespaces with `devcontainer.json`
4. Most likely: hybrid — CLI for interactive work, one of the above for async tasks

## Platform Options Summary

| Platform | Browser | Parallel | Cost | Setup | Status |
|---|---|---|---|---|---|
| Claude Code CLI (local) | Via [playwright-cli](https://github.com/microsoft/playwright-cli) | No | Subscription only | Current local env | **Current** |
| Claude Code Remote | No (proxy blocks HTTPS CONNECT) | Yes | Max plan ($100-200/mo) | Minimal | Blocked |
| Docker Sandboxes | Yes (full control) | Yes | VM + API costs | Custom image needed | Future option |
| GitHub Codespaces | Yes (full Ubuntu VM) | Yes | $0.18-2.88/hr + storage | devcontainer.json | Future option |

## Summary

| Finding | Impact | Action |
|---|---|---|
| No `.claude/settings.json` — no permissions, hooks, or restrictions | High | Create with `permissions.deny` and `PreToolUse` hooks |
| No `make setup` bootstrap command | High | Create setup target that checks prerequisites and initializes environment |
| No version pinning for local dev tools | Medium | Add `.tool-versions` or document versions in CLAUDE.md |
| `/selfimprove` references missing CLAUDE.md files | Medium | Create jirachi/CLAUDE.md and zoroark/CLAUDE.md (workstream #13) |
| No MCP server configuration | Medium | Configure PostgreSQL and GitHub MCP servers |
| No devcontainer.json | Medium | Create for reproducibility and future Codespaces use |
| No browser access during development | Medium | Install [playwright-cli](https://github.com/microsoft/playwright-cli) for agent browser access |
| No agent sandbox for parallel work | Low | Defer — CLI works for solo developer, revisit when scaling |

The agent execution environment is functional but unguarded. An agent can do anything — which means it can also break anything. The two highest-leverage improvements are: **`.claude/settings.json`** (defines what agents can and cannot do) and **`make setup`** (lets agents self-provision). After those, MCP servers (especially Playwright for browser access) would most significantly expand agent capabilities. The sandbox question is a future concern that becomes urgent only when parallel agent execution is needed.
