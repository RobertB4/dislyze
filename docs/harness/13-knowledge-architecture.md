# Workstream #13 — Knowledge Architecture

## Current Setup

Knowledge lives in three layers:

| Layer | What exists | Quality |
|---|---|---|
| CLAUDE.md files | 7 files across the repo | Uneven — lugia-backend is excellent, others are thin or stale |
| docs/ directory | Only `docs/harness/` (audit docs) | No general-purpose documentation |
| Custom commands | 6 slash commands in `.claude/commands/` | Good workflow enforcement |

No architecture overview, no onboarding guide, no feature documentation, no ADRs. The root README.md is severely stale.

## What's Already Agent-Friendly

### 1. lugia-backend/CLAUDE.md is genuinely excellent

This is the gold standard for the repository. It covers: 5 essential commands, directory architecture, two-tier testing strategy, validation pattern with examples, error handling conventions, Japanese user-facing messages, enterprise feature implementation (step-by-step), middleware ordering, context management patterns, and database query optimization. An agent working in lugia-backend has real guidance.

### 2. lugia-backend/test/CLAUDE.md teaches integration testing well

Research-first principle, API endpoint discovery workflow, full test file template, database reset patterns, URL construction rules ("NEVER hardcode or guess URLs"), common test patterns with code examples, and 10 best practices. This is the second-best CLAUDE.md in the repo.

### 3. Root CLAUDE.md provides good meta-level guidance

The philosophical principles are genuinely useful: accuracy over speed, task scope discipline, comment philosophy (why not what), code quality guidelines with good/bad examples. These shape agent behavior at a level that specific conventions can't.

### 4. Custom slash commands enforce workflow discipline

The 6 commands in `.claude/commands/` encode the working methodology:
- `/plan` — forces research and multi-option analysis before coding
- `/implement` — re-reads the plan before implementing
- `/integration` and `/e2e` — enforce research-first test writing with explicit scope boundaries
- `/honest` — enables blunt feedback mode
- `/selfimprove` — structured CLAUDE.md self-improvement loop

These are the closest thing to agent workflow guardrails in the repo.

### 5. Code comments follow the "why not what" philosophy where they exist

Sampling across the codebase shows comments are used to explain non-obvious design choices (e.g., the frontend embed pattern in `main.go`, the `Enabled` vs `Active` distinction in enterprise features). The philosophy from root CLAUDE.md is reflected in practice.

## What's NOT Agent-Friendly

### 1. No architecture overview exists anywhere (High Impact)

There is no document explaining how the system fits together. An agent cannot answer basic questions without reading multiple files:
- How do lugia-backend and giratina-backend relate? (Same database, different middleware stacks)
- What is jirachi? (Shared Go library for auth middleware)
- What is zoroark? (Shared Svelte component library)
- What does giratina do vs. lugia? (Admin panel vs. customer app)
- Which services talk to which?

The architecture is only discoverable by reading `go.work`, the root `Makefile`, `package.json` files, and individual CLAUDE.md files — then synthesizing.

**Improvement strategy:** Add an architecture section to the root CLAUDE.md. It doesn't need to be long — a component list with one-line descriptions and the key relationships:

```
## Architecture

- lugia-backend/ — Customer-facing API. Tenant-scoped, RBAC, enterprise features.
- giratina-backend/ — Internal admin API. Cross-tenant operations, no RBAC.
- jirachi/ — Shared Go library. Auth middleware, context utilities, error handling.
- lugia-frontend/ — Customer-facing SvelteKit SPA.
- giratina-frontend/ — Internal admin SvelteKit SPA.
- zoroark/ — Shared Svelte 5 component library (@dislyze/zoroark).
- database/ — Shared PostgreSQL schema. Goose migrations. All three Go modules generate from this.

Both backends share one PostgreSQL database. jirachi provides auth middleware used by both.
Production secrets are in GCP Secret Manager. Local .env files are localhost-only.
```

### 2. jirachi and zoroark have no CLAUDE.md (High Impact)

Two shared libraries that agents interact with constantly have zero documentation:

- **jirachi** exports auth middleware, context utilities (`libctx`), error handling (`errlib`), JWT, rate limiting, responder, SendGrid, and enterprise features. An agent modifying auth behavior needs to understand jirachi's packages, but there's no guide.
- **zoroark** exports Button, Input, Alert, Badge, Spinner, Tooltip, Toast, Slideover, and utility functions (meCache, routing, errors). An agent building UI components needs to know what's available.

The `/selfimprove` command references `@jirachi/CLAUDE.md` and `@zoroark/CLAUDE.md` — both are missing.

**Improvement strategy:** Create CLAUDE.md files for both:
- **jirachi/CLAUDE.md**: Purpose (shared auth/middleware library), package inventory with one-line descriptions, the rule that jirachi must never import from backends, how SQLC queries in jirachi differ from backend queries (auth-only subset).
- **zoroark/CLAUDE.md**: Purpose (shared UI component library), component inventory, how frontends consume it (`file:../zoroark`), that `npm run package` must run before frontend builds, Svelte 5 runes conventions.

### 3. Root README.md is actively misleading (High Impact)

The README shows a two-component project (`lugia-backend/`, `lugia-frontend/`) when the repo now has six sub-projects plus infrastructure, database, and mock services. Commands reference wrong directory names (`cd backend`). No mention of Docker, PostgreSQL setup, or the database layer.

A new agent (or the `/selfimprove` loop) reading this file first would build a fundamentally wrong mental model.

**Improvement strategy:** Rewrite the README. It should cover: what the project is, the component topology, prerequisites, how to get running (`make initdb && make dev`), and links to sub-project CLAUDE.md files. Keep it short — the CLAUDE.md files handle the depth.

### 4. No onboarding path from zero to running (High Impact)

There is no document that takes an agent (or developer) from a fresh clone to a running system. The required steps are:
1. Install Go, Node.js, PostgreSQL, Docker, goose, sqlc, air, golangci-lint, gotestfmt
2. Start PostgreSQL
3. `make initdb` (migrate + seed)
4. `make dev` (start all services)

This knowledge is scattered across Makefiles, `.env` files, and CI workflows. An agent would need to discover each step through exploration.

**Improvement strategy:** This overlaps with [#2 Development environment](02-dev-environment.md) — the proposed `make setup` script. Document the steps in the README or a dedicated setup guide. Once `make setup` exists, the onboarding path is: `make setup && make dev`.

### 5. giratina-frontend CLAUDE.md is a copy-paste of lugia's (Medium Impact)

The file is nearly identical to lugia-frontend's CLAUDE.md with only the prettierrc path changed. An agent working in giratina-frontend has no idea it's an admin panel, what features it manages, or how it differs from the customer-facing app.

This was flagged in [#6 SvelteKit frontend](06-sveltekit-frontend.md#7-frontend-claudemd-files-are-thin-and-near-identical-low-impact). The recommendation: describe giratina's scope abstractly ("admin panel for internal operators managing tenants across the system").

**Improvement strategy:** Rewrite giratina-frontend/CLAUDE.md with its specific purpose, scope, and how it differs from lugia-frontend (simpler nav, no RBAC, no enterprise features, cross-tenant operations).

### 6. database/CLAUDE.md is barely useful (Medium Impact)

It lists table names but provides no operational information: no migration commands, no goose conventions, no explanation of how SQLC generation works, no schema conventions, no guidance on adding new tables or columns.

**Improvement strategy:** Flesh out with: available commands (`make migrate`, `make seed`, `make initdb`), goose migration file structure (with up/down example), table naming conventions, the relationship between `database/migrations/` and the three `sqlc.yaml` configs, and a note about the `queries_pregeneration/` naming.

### 7. Root CLAUDE.md has no commands or links (Medium Impact)

An agent starting at the root has no idea what commands are available (`make dev`, `make initdb`, etc.) and no links to sub-project CLAUDE.md files. The root file provides philosophy but no operations.

**Improvement strategy:** Add a "Commands" section listing the root Makefile targets. Add a "Sub-project Documentation" section linking to each CLAUDE.md. The root file should be the entry point that routes agents to the right place.

### 8. No feature documentation (Medium Impact)

The auth flow (JWT + refresh tokens), RBAC system, SSO integration, IP whitelist, user invitation flow, email change flow, and password reset flow are undocumented. Feature knowledge exists only in code and partially in lugia-backend's CLAUDE.md (enterprise features pattern).

An agent implementing a new feature that touches auth or RBAC has no high-level understanding of how these systems work — they must reverse-engineer from code.

**Improvement strategy:** Feature documentation doesn't need to be exhaustive. A `docs/features/` directory with one-page overviews of the key flows (auth lifecycle, RBAC model, SSO integration) would give agents the mental model they need. Prioritize auth (JWT lifecycle, refresh token rotation) and RBAC (roles, permissions, the fallback role pattern) since these are the most complex and most likely to be touched.

### 9. No ADRs — architectural decisions are undocumented (Low Impact)

Decisions like "POST for all mutations," "SQLC over raw queries," "Felte for forms," "static adapter (SPA mode)," and "Japanese for all UI text" are made but never recorded. An agent can't understand the reasoning behind these choices — they can only see the current state.

**Improvement strategy:** ADRs are valuable for harness engineering because they teach agents the "why" behind conventions. Start lightweight: when implementing harness improvements, record the decision in a `docs/adr/` directory. Format: title, context, decision, consequences. Don't retroactively document everything — capture decisions going forward.

### 10. Frontend CLAUDE.md files miss critical conventions (Low Impact)

Multiple conventions identified in [#6 SvelteKit frontend](06-sveltekit-frontend.md) are not in any CLAUDE.md:
- The promise-resolution-in-Layout pattern
- `$effect` as last resort
- Felte stores use `$` prefix (not runes)
- Props destructured as `let { data: pageData }`
- Never `await` in load functions
- `$app/state` not `$app/stores`
- Japanese for all UI text

**Improvement strategy:** Add these to lugia-frontend's CLAUDE.md. A "New Page Recipe" section would be particularly high-value — a step-by-step template an agent can follow to create a new route.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No architecture overview | High | Add component topology to root CLAUDE.md |
| jirachi and zoroark have no CLAUDE.md | High | Create CLAUDE.md for both shared libraries |
| Root README.md is stale and misleading | High | Rewrite with current topology and setup |
| No onboarding path from zero to running | High | Document in README, implement `make setup` |
| giratina-frontend CLAUDE.md is copy-paste | Medium | Rewrite with admin-specific scope |
| database/CLAUDE.md barely useful | Medium | Add commands, conventions, SQLC relationship |
| Root CLAUDE.md has no commands or links | Medium | Add commands section and sub-project links |
| No feature documentation | Medium | Create one-page overviews for auth, RBAC, SSO |
| No ADRs | Low | Start recording decisions going forward |
| Frontend CLAUDE.md missing conventions | Low | Add patterns from workstream #6 audit |

The knowledge architecture has one excellent node (`lugia-backend/CLAUDE.md`) surrounded by thin or missing documentation. The root is weak (no architecture, no commands, no links), two shared libraries have nothing, and the README is actively misleading. The highest-leverage improvements are: **architecture overview in root CLAUDE.md** (gives agents the system map), **CLAUDE.md for jirachi and zoroark** (documents the libraries agents use most), and **rewriting the README** (fixes the first thing any agent reads).
