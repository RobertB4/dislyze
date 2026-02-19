# Harness Engineering Environment — Task List

This document tracks the workstreams required to set up a fully harnessed, agent-first development environment.

## Tech Stack Decision

- **Backend**: Go (Chi router, SQLC, PostgreSQL)
- **Frontend**: SvelteKit (Svelte 5, TypeScript)
- **Contract Layer**: OpenAPI spec as source of truth, generating Go server code + TypeScript client
- **Database**: PostgreSQL with SQLC for typed query generation

## General Note

We are not starting from zero. This repository already has a working Go backend, SvelteKit frontend, PostgreSQL database, CI/CD, and more. The goal is not to rebuild everything from scratch but to evaluate and improve what we have through an agent-first lens.

For every workstream below, the approach is:

1. **Audit**: Is our current setup easy for agents to navigate and understand?
2. **Identify gaps**: Which parts are agent-friendly already, and which are not?
3. **Improve**: How do we change the parts that are hard for agents to become easier for agents?

The output of each workstream may range from "this is already good, no changes needed" to "this needs significant restructuring." We assess first, then act.

## Workstreams

### Environment — The codebase agents work in

| # | Workstream | Description |
|---|---|---|
| 1 | Repository & monorepo scaffolding | Directory structure, git setup, Makefile |
| 2 | Development environment | Docker Compose for PostgreSQL, local dev scripts, init.sh |
| 3 | Go backend scaffold | Server, Chi router, domain structure, middleware skeleton |
| 4 | Database layer | PostgreSQL schema, migrations tooling, SQLC generation pipeline |
| 5 | OpenAPI contract layer | Spec format, Go server code generation, TypeScript client generation |
| 6 | SvelteKit frontend scaffold | Project setup, generated API client integration, component conventions |
| 7 | Generation pipeline | Single command that regenerates everything (SQLC + OpenAPI server + OpenAPI client) |

### Enforcement — The constraints that keep agents on track

| # | Workstream | Description |
|---|---|---|
| 8 | Testing infrastructure | Go test patterns, Vitest for frontend, Playwright for e2e |
| 9 | Custom linters | Go linters (dependency direction, naming, domain isolation), Svelte linters. Error messages that teach the fix. |
| 10 | Structural tests | Architecture validation tests (import rules, file conventions, domain boundaries) |
| 11 | CI/CD pipeline | GitHub Actions with strict gate ordering: generate → lint → typecheck → structural tests → unit tests → e2e |
| 12 | Security boundaries & secrets management | What agents can access, API key handling, database credentials, blast radius containment |

### Knowledge — The documentation that guides agents

| # | Workstream | Description |
|---|---|---|
| 13 | Knowledge architecture | CLAUDE.md, docs/ structure, architecture map, beliefs, quality grades, design doc templates |
| 14 | Feature tracking system | feature-list.json structure and conventions |
| 15 | Version control strategy | Branch naming, PR conventions, commit message format, merge rules |
| 16 | Human escalation protocol | Clear rules for when an agent must stop and ask a human |

### Agents — The infrastructure for agent execution

| # | Workstream | Description |
|---|---|---|
| 17 | Agent execution environment | CLI tool setup (Claude Code / Codex), sandboxing, permissions, available tools |
| 18 | Agent workflow & orchestration | How agents pick up tasks, track progress across sessions, know when they're done |
| 19 | Agent review pipeline | Agents reviewing other agents' PRs, self-review loops, merge criteria |
| 20 | Agent-specific tooling | Utility scripts: screenshot capture, browser inspection, filtered test running, dev server management |
| 21 | Multi-agent roles & coordination | Defining agent types (coding, review, doc-gardening, quality-grading, security) and how they interact |

### Meta — The system that makes the system better over time

| # | Workstream | Description |
|---|---|---|
| 22 | Agent retrospective / self-improvement loop | After every task, agents reflect abstractly on what they struggled with and propose harness improvements. Proposals are collected, reviewed, and applied. |
| 23 | Observability & metrics | Agent success rate, iterations before completion, common failure modes, time-to-merge. Measures whether harness improvements actually help. |
| 24 | Garbage collection agents | Doc-gardening agent, quality-grading agent, stale code detection. Fights entropy over time. |

---

## Workstream Audits

Detailed findings for each workstream live in `docs/harness/`. HARNESS.md is the overview — not the encyclopedia.

| # | Workstream | Audit | Status |
|---|---|---|---|
| 1 | Repository & monorepo scaffolding | [docs/harness/01-repo-scaffolding.md](docs/harness/01-repo-scaffolding.md) | Audited |
| 2 | Development environment | [docs/harness/02-dev-environment.md](docs/harness/02-dev-environment.md) | Audited |
| 3 | Go backend scaffold | [docs/harness/03-go-backend.md](docs/harness/03-go-backend.md) | Audited |
| 4 | Database layer | [docs/harness/04-database-layer.md](docs/harness/04-database-layer.md) | Audited |
| 5 | OpenAPI contract layer | [docs/harness/05-openapi-contract-layer.md](docs/harness/05-openapi-contract-layer.md) | Audited |
| 6 | SvelteKit frontend scaffold | [docs/harness/06-sveltekit-frontend.md](docs/harness/06-sveltekit-frontend.md) | Audited |
| 7 | Generation pipeline | [docs/harness/07-generation-pipeline.md](docs/harness/07-generation-pipeline.md) | Audited |
| 8 | Testing infrastructure | [docs/harness/08-testing-infrastructure.md](docs/harness/08-testing-infrastructure.md) | Audited |
| 9 | Custom linters | [docs/harness/09-custom-linters.md](docs/harness/09-custom-linters.md) | Audited |
| 10 | Structural tests | [docs/harness/10-structural-tests.md](docs/harness/10-structural-tests.md) | Audited |
| 11 | CI/CD pipeline | [docs/harness/11-cicd-pipeline.md](docs/harness/11-cicd-pipeline.md) | Audited |
| 12 | Security boundaries & secrets management | [docs/harness/12-security-boundaries.md](docs/harness/12-security-boundaries.md) | Audited |
| 13 | Knowledge architecture | [docs/harness/13-knowledge-architecture.md](docs/harness/13-knowledge-architecture.md) | Audited |
| 14 | Feature tracking system | [docs/harness/14-feature-tracking.md](docs/harness/14-feature-tracking.md) | Audited |
| 15 | Version control strategy | [docs/harness/15-version-control-strategy.md](docs/harness/15-version-control-strategy.md) | Audited |
| 16 | Human escalation protocol | [docs/harness/16-human-escalation-protocol.md](docs/harness/16-human-escalation-protocol.md) | Audited |
| 17 | Agent execution environment | [docs/harness/17-agent-execution-environment.md](docs/harness/17-agent-execution-environment.md) | Audited |
| 18 | Agent workflow & orchestration | [docs/harness/18-agent-workflow-orchestration.md](docs/harness/18-agent-workflow-orchestration.md) | Audited |
| 19 | Agent review pipeline | [docs/harness/19-agent-review-pipeline.md](docs/harness/19-agent-review-pipeline.md) | Audited |
| 20 | Agent-specific tooling | [docs/harness/20-agent-specific-tooling.md](docs/harness/20-agent-specific-tooling.md) | Audited |
| 21 | Multi-agent roles & coordination | [docs/harness/21-multi-agent-roles-coordination.md](docs/harness/21-multi-agent-roles-coordination.md) | Audited |
| 22 | Agent retrospective / self-improvement loop | [docs/harness/22-agent-retrospective.md](docs/harness/22-agent-retrospective.md) | Audited |
| 23 | Observability & metrics | [docs/harness/23-observability-metrics.md](docs/harness/23-observability-metrics.md) | Audited |
| 24 | Garbage collection agents | [docs/harness/24-garbage-collection-agents.md](docs/harness/24-garbage-collection-agents.md) | Audited |

---

## Implementation Plan

All 24 workstreams are audited. The prioritized implementation plan lives at [docs/harness/implementation-plan.md](docs/harness/implementation-plan.md).

Implementation order: **Tier 0** (measurement foundation) → **Tier 1** (quick wins) → **Tier 2** (medium effort, high impact) → **Tier 3** (medium effort, medium impact).

---

## References

- [Harness Engineering | OpenAI](https://openai.com/index/harness-engineering/)
- [Harness Engineering | Martin Fowler](https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html)
- [Effective Harnesses for Long-Running Agents | Anthropic](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- [The Importance of Agent Harness in 2026 | Phil Schmid](https://www.philschmid.de/agent-harness-2026)
- [Harness Engineering Is Not Context Engineering | Substack](https://mtrajan.substack.com/p/harness-engineering-is-not-context)
- [Agent-Native Engineering | General Intelligence Company](https://www.generalintelligencecompany.com/writing/agent-native-engineering)
- [My AI Adoption Journey | Mitchell Hashimoto](https://mitchellh.com/writing/my-ai-adoption-journey)
