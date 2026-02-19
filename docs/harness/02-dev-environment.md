# Workstream #2 — Development Environment

## Current Setup

### Prerequisites (implicit, not documented)
- Go 1.24+
- Node.js + npm
- PostgreSQL running locally on port 5432
- Docker (for integration/e2e tests)
- `air` (Go hot reload) installed at `$(HOME)/go/bin/air`
- `goose` (migrations) installed globally
- `gotestfmt` installed globally
- `sqlc` installed globally
- `golangci-lint` installed globally (or via `make install`)

### How local dev works
- `make dev` at the root starts 6 services in parallel via `make -j6`:
  - lugia-backend (air hot reload, port 3001)
  - lugia-frontend (SvelteKit dev server, port 3000)
  - giratina-backend (air hot reload, port 4001)
  - giratina-frontend (SvelteKit dev server, port 4000)
  - sendgrid-mock (Node.js, port 7000)
  - keycloak-mock (local Keycloak binary, port 7001)

### Database setup
- `make migrate` — runs goose migrations against local PostgreSQL
- `make seed` — runs seed.sql
- `make initdb` — drops, migrates, and seeds (full reset)
- PostgreSQL is expected to already be running (no Docker Compose for local dev DB)

### Environment variables
- `.env` files in each backend directory (committed to git)
- `.env.sensitive` in lugia-backend (contains real Anthropic API key — committed to git)
- `.env` in giratina-frontend (one variable: `PUBLIC_LUGIA_FRONTEND_URL`)
- `.env` and `.env.e2e` in lugia-frontend (both empty)

### Testing environments
- **Integration tests**: Docker Compose per backend (`test/docker-compose.integration.yml`), spins up PostgreSQL + sendgrid mock + backend container. Port ranges: 13xxx (lugia), 14xxx (giratina presumably).
- **E2E tests**: Docker Compose in lugia-frontend (`test/docker-compose.e2e.yml`), spins up PostgreSQL + sendgrid + keycloak + backend + frontend + Playwright container. Port range: 2xxxx.
- Each test environment uses its own port ranges to avoid conflicts.

### Hot reload
- Air watches both the backend directory and jirachi (shared library) for changes
- `root = ".."` in both `.air.toml` files — Air runs from the monorepo root
- Frontend uses SvelteKit's built-in Vite dev server

## What's Already Agent-Friendly

1. **Single command to start everything.** `make dev` is all an agent needs to know. No multi-step startup ritual.

2. **Clear Makefile commands.** `make migrate`, `make seed`, `make initdb` — an agent can reset the database state predictably.

3. **Air watches shared library changes.** When an agent modifies jirachi, both backends auto-reload. No manual restart needed.

4. **Test environments are fully containerized.** Integration and e2e tests don't depend on local state beyond Docker. An agent can run `make test-integration` without worrying about local PostgreSQL config.

5. **Port separation between test environments.** Integration tests (13xxx), e2e tests (2xxxx), and local dev (3xxx/4xxx) don't conflict. An agent can run local dev and tests simultaneously.

## What's NOT Agent-Friendly

### 1. No single setup script for a fresh clone (High Impact)

There is no `make setup` or `init.sh` that installs all prerequisites and gets the environment running from zero. An agent (or new developer) cloning this repo has to figure out:
- Install Go, Node, PostgreSQL, Docker
- Install air, goose, gotestfmt, sqlc, golangci-lint
- Start PostgreSQL
- Run `make migrate && make seed`
- Then `make dev`

This is especially important for agents because they need to know exactly how to get a working environment. If an agent's dev server isn't running, it can't verify its own changes.

**Improvement strategy:** Create a `make setup` target (or `scripts/setup.sh`) that:
- Checks for required tools and tells you what's missing
- Installs what it can (Go tools via `go install`)
- Runs migrations and seeds
- Validates everything works

Also document the full setup flow in CLAUDE.md with a "Getting Started" section.

### 2. PostgreSQL is assumed to be running locally — no Docker Compose for local dev (Medium Impact)

Integration and e2e tests spin up PostgreSQL in Docker, but local dev assumes PostgreSQL is already running on localhost:5432. There's no `docker-compose.yml` at the root for local development.

An agent can't start PostgreSQL on its own if it's not running. This creates a hidden dependency.

**Improvement strategy:** Add a root `docker-compose.yml` for local dev that starts PostgreSQL (and optionally the mock services). This way `make dev` could first ensure the database is running.

### 3. Agents can read `.env.sensitive` despite it being gitignored (High Impact — Security)

`lugia-backend/.env.sensitive` is correctly gitignored, but nothing prevents an agent from reading it. During this audit, the agent read the file and loaded a real API key into its context window. The key has since been rotated.

This is a harness engineering gap: **gitignore protects against committing secrets, but does not protect against agent access.** Agents need separate access controls.

**Improvement strategy:** See workstream #12 (Security boundaries). Specific mechanisms for Claude Code:
- `permissions.deny` in `.claude/settings.json` — blocks Claude Code from reading files matching specified patterns (e.g., `Read(./.env.sensitive)`)
- `PreToolUse` hooks — pre-read hooks that inspect and deny access to sensitive files with custom error messages

See [#12 Security boundaries](12-security-boundaries.md) for the full assessment and recommended configuration.

### 4. Tool installation is scattered and implicit (Medium Impact)

Required Go tools are installed differently:
- `golangci-lint` via a curl script in `make install`
- `air` is assumed to be at `$(HOME)/go/bin/air` (no install target)
- `goose`, `gotestfmt`, `sqlc` — no install targets at all
- `deadcode` — referenced in Makefile but no install instruction

An agent running `make lint` will get a confusing error if golangci-lint isn't installed, with no guidance on how to fix it.

**Improvement strategy:** Add a `make tools` target that installs all required Go tools:
```makefile
tools:
    go install github.com/air-verse/air@latest
    go install github.com/pressly/goose/v3/cmd/goose@latest
    go install gotest.tools/gotestfmt/v2/cmd/gotestfmt@latest
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    go install golang.org/x/tools/cmd/deadcode@latest
    # golangci-lint needs special install
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.1.6
```

### 5. `make dev` output is chaotic with 6 services (Low Impact)

`make -j6` runs all 6 services in parallel, interleaving their stdout. When something fails, it's hard to tell which service errored. An agent parsing terminal output would struggle to isolate a specific service's error.

**Improvement strategy:** Consider using a process manager like `overmind` or `foreman` with a `Procfile`, which gives each service its own labeled output stream. Or at minimum, prefix each service's output. This is low priority — it's more a developer experience issue than an agent blocker.

### 6. No health check or readiness verification after startup (Medium Impact)

After `make dev`, there's no way to programmatically verify that all services are up and healthy. An agent would need to know to hit `http://localhost:3001/health` (backend) and check that the frontend is serving.

For harness engineering, the Anthropic article recommends an `init.sh` that starts services and runs a smoke test before any coding begins.

**Improvement strategy:** Add a `make dev-ready` or `make health` target that:
- Starts services (or assumes they're started)
- Polls health endpoints until all services respond
- Exits 0 when everything is ready, or exits 1 with a clear error

### 7. Three separate Docker Compose files for testing with duplicated config (Medium Impact)

Test infrastructure is split across:
- `lugia-backend/test/docker-compose.integration.yml`
- `giratina-backend/test/docker-compose.integration.yml` (similar)
- `lugia-frontend/test/docker-compose.e2e.yml`

Each duplicates PostgreSQL config, sendgrid mock config, environment variables, and SAML certificates. The SAML certificates in particular are copy-pasted across all three files (50+ lines each).

**Improvement strategy:**
- Extract shared env vars and certificates to files that Docker Compose can reference (e.g., `test/shared.env`, `test/saml-certs/`)
- Consider a single `test/docker-compose.yml` with profiles for integration vs. e2e
- At minimum, extract the SAML certificates to files instead of inline in YAML

### 8. No documentation of port assignments (Low Impact)

Port allocation across environments:
- Local dev: 3000/3001 (lugia), 4000/4001 (giratina), 7000 (sendgrid), 7001 (keycloak)
- Integration tests: 13001 (lugia), 15432 (postgres), 17000 (sendgrid)
- E2E tests: 23000/23001 (lugia), 25432 (postgres), 27000 (sendgrid), 27001 (keycloak)

This is a well-designed system (clear port ranges per environment) but it's undocumented. An agent adding a new service or test environment would have to reverse-engineer the convention.

**Improvement strategy:** Document port conventions in `docs/architecture.md` or a `docs/ports.md`.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No setup script for fresh clone | High | Create `make setup` / `scripts/setup.sh` |
| Agents can read `.env.sensitive` despite gitignore | High | Add `permissions.deny` in `.claude/settings.json` (workstream #12) |
| No Docker Compose for local dev DB | Medium | Add root `docker-compose.yml` for PostgreSQL |
| Tool installation scattered/implicit | Medium | Add `make tools` target |
| No health check after startup | Medium | Add `make health` or `make dev-ready` target |
| Duplicated Docker Compose config | Medium | Extract shared config, consider profiles |
| `make dev` output interleaved | Low | Consider process manager (overmind/Procfile) |
| Port assignments undocumented | Low | Document port conventions |

The highest-leverage changes are the **setup script** and fixing the **committed API key**. A setup script is the single most important thing for agent-first development — an agent needs to be able to get a working environment deterministically.
