# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Full-stack multi-tenant SaaS application: Go backends (Chi, SQLC, PostgreSQL) + SvelteKit frontends (Svelte 5, TypeScript, Tailwind).

Key capabilities: JWT auth, multi-tenant architecture, RBAC, SSO (SAML/OIDC via Keycloak), IP whitelisting, user invitations, password reset, email change verification, rate limiting.

## Architecture

```
                    ┌──────────────┐
                    │   database/  │  PostgreSQL schema + migrations (goose)
                    └──────┬───────┘
                    ┌──────┴───────┐
                    │   jirachi/   │  Shared Go library (auth, jwt, errlib, etc.)
                    └──┬───────┬───┘
          ┌────────────┴─┐   ┌─┴──────────────┐
          │lugia-backend │   │giratina-backend │  Go HTTP servers
          │  (customer)  │   │  (internal admin)│
          └──────┬───────┘   └──────┬──────────┘
                    ┌──────────────┐
                    │   zoroark/   │  Shared Svelte 5 component library
                    └──┬───────┬───┘
        ┌──────────────┴─┐   ┌─┴────────────────┐
        │lugia-frontend  │   │giratina-frontend  │  SvelteKit apps
        └────────────────┘   └───────────────────┘
```

| Directory            | What it is                                                                                 | Language   |
| -------------------- | ------------------------------------------------------------------------------------------ | ---------- |
| `database/`          | Schema migrations (goose), seed data, drop script                                          | SQL        |
| `jirachi/`           | Shared Go library — auth, authz, context, error handling, JWT, rate limiting, SQLC queries | Go         |
| `lugia-backend/`     | Customer-facing API server                                                                 | Go         |
| `giratina-backend/`  | Internal admin API server                                                                  | Go         |
| `zoroark/`           | Shared Svelte 5 component library (Button, Input, Alert, Toast, etc.)                      | Svelte/TS  |
| `lugia-frontend/`    | Customer-facing SvelteKit app                                                              | Svelte/TS  |
| `giratina-frontend/` | Internal admin SvelteKit app                                                               | Svelte/TS  |
| `infrastructure/`    | Pulumi IaC for GCP deployment                                                              | TypeScript |
| `sendgrid-mock/`     | Mock SendGrid server for dev                                                               | Node.js    |
| `keycloak-mock/`     | Mock Keycloak server for SSO dev                                                           | Shell      |

### Dependency direction

- Backends depend on `jirachi/` — never the other way around
- Frontends depend on `zoroark/` — never the other way around
- `jirachi/` and `zoroark/` must NOT depend on any backend or frontend module
- `database/` is standalone — migrations are shared by all Go modules

## Essential commands

```bash
make dev              # Start all services (6 processes)
make generate         # Regenerate all SQLC across all modules
make migrate          # Run database migrations
make initdb           # Drop + migrate + seed (destructive)
```

**Test accounts:** use accounts from `database/seed.sql` for browser testing with playwright-cli. Check the seed file for emails and passwords.

## General guidelines

- **Accuracy over speed** — understand the problem and plan before writing code. Ask clarifying questions.
- **Verify before explaining** — when asked about existing behavior, read the code/config first. Don't reason from memory or plans — the implementation may differ.
- **Verify changes work** — test happy path, edge cases, and failure modes. Fix root causes, not symptoms.
- **Root cause over workaround** — fix the underlying cause, not the surface symptom. Flag it if the root cause fix is significantly more work.
- **Task scope** — focus exclusively on the task at hand. Note unrelated improvements as comments, don't implement them.
- **Comments explain WHY, not WHAT** — see `docs/conventions.md` for examples.
- **Locality of behavior** — code that belongs together should live together. Only separate with good reason.
- **Follow existing patterns** — study the codebase first. Use existing types, match existing interfaces, follow naming conventions.
- **Design for concurrent access** — when introducing shared artifacts (files, configs, state), consider who reads and writes them. If multiple processes might write to the same file, externalize mutable state to a coordination system. Keep shared files read-mostly.
- **Prefer simplicity** — simple interfaces, direct solutions, type safety, minimal dependencies.

## Key rules

### Generated code boundaries

Files in `queries/` directories are generated by SQLC. **Never hand-edit them.** Edit SQL in `queries_pregeneration/`, run `make generate`, commit both.

### Shared resource blast radius

| Resource               | Consumed by                       | Impact                                           |
| ---------------------- | --------------------------------- | ------------------------------------------------ |
| `database/migrations/` | All Go modules                    | Schema changes affect all backends               |
| `jirachi/`             | lugia-backend, giratina-backend   | Library changes affect both backends             |
| `zoroark/`             | lugia-frontend, giratina-frontend | Component changes affect both frontends          |
| `database/seed.sql`    | Local dev setup                   | Seed changes can break local dev for all modules |

When changing a shared resource, verify all consumers still work by running their check commands (see each service's CLAUDE.md).

## Definition of Done

1. **Behavior verified**: The feature works — confirmed by running the service's verify command AND by using `playwright-cli` to verify frontend changes in the browser. "It compiles" is not verification.
2. **Tests written and passing**: New behavior has corresponding tests. Tests run and pass — not just written, but executed. Unit tests, integration tests, e2e tests.
3. **Each step verified independently**: After completing each step in the task, verify it works before moving on. Do not batch all verification to the end.

## Escalation protocol

**Stop and ask** when: ambiguous requirements, shared resource changes you're unsure about, security decisions, destructive operations, scope creep, or you're blocked after two attempts.

**Don't ask** when: the task is well-defined, you're following existing patterns, or the change is contained to a single module.

## Deeper documentation

This root file is read every session — keep it concise. Detailed information belongs in deeper docs, read on-demand when working on the relevant module or task. When adding documentation, consider when agents need it, not just where it logically belongs.

| Document                      | What it covers                                                         |
| ----------------------------- | ---------------------------------------------------------------------- |
| `docs/features/`              | Per-feature docs: design intent, interactions, non-obvious constraints |
| `docs/conventions.md`         | Detailed coding conventions with examples                              |
| `database/CLAUDE.md`          | Schema migrations, seed data, goose conventions                        |
| `lugia-backend/CLAUDE.md`     | Customer backend: handlers, middleware, enterprise features            |
| `giratina-backend/CLAUDE.md`  | Admin backend: handlers, tenant management                             |
| `lugia-frontend/CLAUDE.md`    | Customer frontend: routes, components, fetch patterns                  |
| `giratina-frontend/CLAUDE.md` | Admin frontend: routes, components                                     |
| `jirachi/CLAUDE.md`           | Shared Go library: packages, key rules                                 |
| `zoroark/CLAUDE.md`           | Shared UI library: components, build process                           |
