# Workstream #1 — Repository & Monorepo Scaffolding

## Current Structure

```
dislyze/
├── .claude/                  # Claude Code custom commands
├── .github/                  # CI/CD workflows
├── .vscode/                  # Editor settings
├── database/                 # Shared PostgreSQL migrations + seed
├── giratina-backend/         # Go — internal admin API
├── giratina-frontend/        # SvelteKit — internal admin UI
├── infrastructure/           # Pulumi IaC (TypeScript, GCP)
├── jirachi/                  # Shared Go library (auth, JWT, errors, rate limiting)
├── keycloak-mock/            # OIDC/SSO mock for local dev
├── lugia-backend/            # Go — customer-facing API
├── lugia-frontend/           # SvelteKit — customer-facing UI
├── sendgrid-mock/            # Email mock for local dev
├── zoroark/                  # Shared Svelte 5 component library
├── go.work                   # Go workspace (jirachi + lugia + giratina)
├── Makefile                  # Root orchestration (dev, migrate, seed)
├── CLAUDE.md
├── HARNESS.md
└── README.md
```

Two independent applications (lugia = customer-facing, giratina = internal admin) sharing a database, a Go library (jirachi), and a Svelte UI library (zoroark).

## What's Already Agent-Friendly

1. **Flat top-level structure.** Every major component is a top-level directory. An agent can `ls` the root and immediately see all the pieces. No deeply nested indirection to find things.

2. **One domain per directory in `features/`.** Both backends organize handlers as `features/auth/`, `features/users/`, `features/roles/`, `features/ip_whitelist/`. An agent can look at one domain to learn the pattern, then replicate it.

3. **One file per endpoint.** `login.go`, `signup.go`, `forgot_password.go` — extremely granular. An agent working on password reset never needs to read auth login code. Small files = less context needed.

4. **CLAUDE.md files at multiple levels.** Root, both backends, both frontends, database, test directory. Agents get scoped context wherever they are.

5. **Consistent patterns across both apps.** Giratina follows the same handler/service/test structure as lugia. An agent that understands one can work on the other.

6. **Makefile as entry point.** `make dev`, `make test-unit`, `make test-integration`, `make lint`, `make sqlc` — clear, discoverable commands.

7. **SQLC for all database access.** Predictable workflow: write SQL in `queries_pregeneration/`, run `make sqlc`, get typed Go. No ambiguity about where DB access happens.

8. **Co-located tests.** Unit tests sit next to the code they test. Integration tests mirror the domain structure. Agent doesn't have to hunt for related tests.

## What's NOT Agent-Friendly

### 1. Pokemon naming carries zero semantic meaning (High Impact)

An agent seeing `jirachi/` for the first time has no idea it's a shared auth library. Same for `lugia` (customer app), `giratina` (admin app), `zoroark` (UI components). Every new agent session starts with a "what is this?" overhead.

Compare:
- `jirachi/` → agent must read code to understand purpose
- `shared-go/` or `platform/` → purpose is immediately clear

This matters because agents read CLAUDE.md for context, but directory names are the first signal they encounter when navigating. Semantic names reduce the context needed to orient.

**Improvement strategy:** Rename to semantic names. Possible mapping:
- `lugia-backend/` → `app-backend/` or `backend/`
- `lugia-frontend/` → `app-frontend/` or `frontend/`
- `giratina-backend/` → `admin-backend/`
- `giratina-frontend/` → `admin-frontend/`
- `jirachi/` → `shared-go/` or `platform/`
- `zoroark/` → `shared-ui/` or `ui-kit/`

This is a disruptive change (affects imports, CI paths, Docker, go.work, go.mod) but a one-time cost with permanent benefit.

### 2. No architecture documentation beyond CLAUDE.md (High Impact)

There is no document that explains:
- The relationship between lugia and giratina
- Why there are two apps sharing one database
- The dependency rules (what can import what)
- The data flow from request → middleware → handler → service → DB
- The security model (how JWT auth works across both apps, how giratina can impersonate lugia tenants)

An agent working on giratina's "log in to tenant" feature has to piece together the cross-app JWT relationship by reading code across multiple directories. A single architecture doc would eliminate this discovery cost.

**Improvement strategy:** Create `docs/architecture.md` that maps the full system. This overlaps with workstream #13 (Knowledge Architecture) but the need is identified here.

### 3. Duplicated code between applications (Medium Impact)

- `lib/fetch.ts` is identical in both frontends but not extracted to zoroark
- Both backends have similar `lib/db/`, `lib/config/` patterns
- Makefiles are largely duplicated

When an agent improves the fetch wrapper in lugia-frontend, it won't know to update giratina-frontend. Duplicated code means duplicated agent work and drift risk.

**Improvement strategy:**
- Extract `lib/fetch.ts` to zoroark (shared UI library)
- Evaluate whether more backend utilities should move to jirachi
- Consider a shared Makefile template or root-level Makefile targets that delegate

### 4. `queries_pregeneration/` is a non-standard name (Medium Impact)

This directory holds the raw SQL that SQLC compiles. The name is long, unusual, and doesn't follow any convention an agent would recognize. An agent looking for "where do I write SQL queries" wouldn't guess `queries_pregeneration/`.

**Improvement strategy:** Rename to `sql/` or `queries/sql/`. Then the generated code in `queries/` makes sense as the output. The pair `sql/` → `queries/` is immediately understandable.

### 5. `lib/` is a grab bag of unrelated concerns (Medium Impact)

`lib/` contains infrastructure (`db/`, `config/`), domain-adjacent logic (`authz/`, `middleware/`), and generic utilities (`iputils/`, `conversions/`, `pagination/`, `search/`). An agent looking for "where does authorization logic live" has to scan through unrelated utilities.

This isn't critical because the files are small and well-named, but it slightly increases navigation cost.

**Improvement strategy:** Could reorganize into:
- `internal/platform/` — db, config, middleware (infrastructure)
- `internal/authz/` — authorization logic (domain-adjacent)
- `internal/utils/` — iputils, conversions, pagination, search (generic utilities)

Or keep `lib/` but accept it as a known trade-off. The current flat structure in `lib/` is at least easy to scan.

### 6. No API contract between frontend and backend (High Impact)

Frontend types (the `Me` type in zoroark, `types.ts` in routes) are manually written and can drift from backend Go structs. There's no OpenAPI spec, no generated TypeScript client. An agent changing a Go response struct won't know to update the frontend type.

**Improvement strategy:** This is workstream #5 (OpenAPI contract layer). Flagged here because it's a structural gap visible at the repo level. The generation pipeline would live at the root level and affect the overall repo structure.

### 7. Infrastructure code is mostly commented out (Low Impact)

`infrastructure/index.ts` has most modules commented out. An agent exploring the repo would be confused by large blocks of dead code with no explanation of why it's commented out or when it will be re-enabled.

**Improvement strategy:** Either delete the commented-out code (it's in git history) or add a clear comment at the top explaining the status. Since infrastructure is out of scope for the harness work, this is low priority.

### 8. No structural tests or custom linters (High Impact)

There are no tests that validate architectural rules:
- "Domains in `features/` cannot import from each other"
- "No direct SQL outside `queries/`"
- "Every domain must have a `handler.go`"
- "Every handler endpoint must have a corresponding test"

Without these, agents can violate architectural rules and the only feedback is human code review. This is an enforcement gap, not a scaffolding gap, but it's visible at the repo structure level.

**Improvement strategy:** This is workstreams #9 and #10. Flagged here as context.

## Summary

| Finding | Impact | Action |
|---|---|---|
| Pokemon naming → no semantic meaning | High | Rename directories to semantic names |
| No architecture documentation | High | Create docs/architecture.md (workstream #13) |
| No API contract between FE/BE | High | Introduce OpenAPI spec (workstream #5) |
| No structural tests / custom linters | High | Add enforcement layer (workstreams #9, #10) |
| Duplicated code across apps | Medium | Extract shared code to jirachi/zoroark |
| `queries_pregeneration/` naming | Medium | Rename to `sql/` |
| `lib/` grab bag | Medium | Consider reorganizing, or accept as-is |
| Infrastructure code commented out | Low | Clean up or annotate |

The highest-leverage change specific to this workstream is **renaming directories to semantic names**. Everything else either overlaps with other workstreams or is a lower-impact improvement.
