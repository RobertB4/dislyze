# Workstream #20 — Agent-Specific Tooling

## Current Setup

There are no agent-specific tools. All tooling is developer-oriented — Makefiles, npm scripts, shell scripts, Docker Compose files. Agents use these same tools, which works but leaves gaps where agent needs differ from human needs.

### Tool inventory

| Category | Tools | Invocation |
|---|---|---|
| Dev server management | Start all 6 services, start individually | `make dev`, `make dev-<service>` |
| Database management | Migrate, seed, full reset | `make migrate`, `make seed`, `make initdb` |
| Backend testing | Unit tests, integration (all), integration (single) | `make test-unit`, `make test-integration`, `make test-integration-single TEST=<name>` |
| Frontend testing | E2E tests (lugia only) | `npm run test-e2e` |
| Code quality | Lint (Go), lint (frontend), type check, dead code | `make lint`, `npm run lint`, `npm run check`, `make deadcode` |
| Code generation | SQLC per module | `make sqlc` (in each Go module) |
| Build | Backend binary, frontend build | `make build`, `npm run build` |
| Mock services | SendGrid email mock, Keycloak SSO mock | Started via `make dev` |
| Email inspection | Fetch captured emails as JSON | `GET http://localhost:7000/json?token=sendgrid` |
| Health checks | Backend health, Keycloak health | `GET /health`, `GET /health/ready` |
| Hot reload | Go backends via air | Configured in `.air.toml` per backend |

No utility scripts directory (`scripts/`, `tools/`, `bin/`) exists anywhere in the repository.

## What's Already Agent-Friendly

### 1. Makefile targets are discoverable and consistent

Both backends follow the same Makefile pattern: `dev`, `test-unit`, `test-integration`, `test-integration-single`, `lint`, `build`, `sqlc`, `deadcode`. An agent can learn the pattern from one backend and apply it to the other. The root Makefile aggregates `dev` targets for all services.

### 2. Single-test execution exists for integration tests

`make test-integration-single TEST=TestCreateRole_Integration` runs one named test in Docker. This is critical for agents — when a test fails, the agent can re-run just that test rather than the entire suite. The target handles the full lifecycle: tear down containers, rebuild, start services, run the specific test.

### 3. SendGrid mock has a JSON inspection endpoint

`GET http://localhost:7000/json?token=sendgrid` returns all captured emails as JSON. An agent writing integration or E2E tests that involve email flows (invitations, password reset, email change, verification) can programmatically inspect email contents to extract tokens and links. This is a genuinely agent-friendly tool — no human would typically curl this endpoint, but an agent can use it to verify email-dependent workflows.

### 4. Health check endpoints enable service readiness verification

Both backends expose `GET /health` returning `200 OK`. An agent can verify services are running before attempting to interact with them. Keycloak mock also has a built-in health endpoint at `/health/ready`.

### 5. Hot reload means agents see changes immediately

Air watches for Go file changes and rebuilds automatically. Vite dev server does the same for frontend changes. An agent making code changes doesn't need to manually restart services — the feedback loop is immediate.

### 6. `make initdb` provides a clean-slate database reset

When an agent needs to start fresh (e.g., after a failed test corrupts data, or when switching between tasks that require different data states), `make initdb` drops everything and rebuilds from migrations + seed data. This is a reliable escape hatch.

## What's NOT Agent-Friendly

### 1. No root-level `make test` or `make check` command (High Impact)

An agent that just finished implementing a feature has no single command to verify everything works. To run a comprehensive check, the agent must know to run multiple commands across multiple directories:

```bash
# What an agent currently must do:
cd lugia-backend && make lint && make test-unit
cd giratina-backend && make lint && make test-unit
cd jirachi && make test-unit
cd lugia-frontend && npm run lint && npm run check
cd giratina-frontend && npm run lint && npm run check
```

There is no `make check-all` or `make verify` that runs lint + type check + unit tests across the entire monorepo.

**Improvement strategy:** Add root-level aggregate targets:

```makefile
lint:
    cd lugia-backend && make lint
    cd giratina-backend && make lint
    cd lugia-frontend && npm run lint
    cd giratina-frontend && npm run lint

check:
    cd lugia-frontend && npm run check
    cd giratina-frontend && npm run check

test-unit:
    cd lugia-backend && make test-unit
    cd giratina-backend && make test-unit
    cd jirachi && make test-unit

verify: lint check test-unit
```

`make verify` becomes the single command an agent runs before reporting "done." This directly supports the Definition of Done from [#18 Workflow](18-agent-workflow-orchestration.md).

### 2. No root-level `make generate` command (High Impact)

SQLC generation is per-module (`make sqlc` in each of lugia-backend, giratina-backend, jirachi). There is no single command to regenerate all generated code. An agent that modifies a database migration must remember to run `make sqlc` in three separate directories.

This was flagged in [#7 Generation pipeline](07-generation-pipeline.md). The proposed solution: a root `make generate` target that runs all code generation in sequence.

**Improvement strategy:** Add to root Makefile:

```makefile
generate:
    cd lugia-backend && make sqlc
    cd giratina-backend && make sqlc
    cd jirachi && make sqlc
```

Extensible for future OpenAPI generation.

### 3. No `make setup` bootstrap command (High Impact)

No command takes a fresh clone to a working environment. An agent encountering a missing tool or broken database has no automated recovery path.

This was flagged in [#2 Development environment](02-dev-environment.md) and [#17 Agent execution environment](17-agent-execution-environment.md). The most-referenced gap across all workstreams.

**Improvement strategy:** Create `make setup` that checks prerequisites and initializes the environment (tracked across workstreams #2, #13, #17).

### 4. No browser/screenshot tool for agents (Medium Impact)

An agent working on frontend code cannot visually verify its changes. There is no screenshot capture utility, no browser inspection script, no way for an agent to "see" what the UI looks like.

Playwright exists but only within the E2E Docker Compose stack — it's not available as a standalone tool for ad-hoc visual verification during development.

**Improvement strategy:** Use [playwright-cli](https://github.com/microsoft/playwright-cli) (`@playwright/cli`). This is a CLI tool from Microsoft built specifically for AI coding agents. It provides browser navigation, clicking, typing, screenshots, PDFs, and page snapshots — all through CLI commands that agents can invoke directly.

Key advantages over an MCP server approach:
- **More token-efficient**: avoids loading large tool schemas and verbose accessibility trees into agent context
- **Session-based**: persistent browser contexts across CLI calls (named sessions via `-s=` flag)
- **Agent skill installation**: `playwright-cli install --skills` installs agent-readable skill files for Claude Code
- **Monitoring dashboard**: `playwright-cli show` opens a visual interface to observe running browser sessions

Installation: `npm install -g @playwright/cli@latest`

This satisfies the hard requirement from [#17](17-agent-execution-environment.md) that agents must have browser access. Include `@playwright/cli` in the tool versions documentation and `make setup` prerequisites.

### 5. No `make stop` or service management (Medium Impact)

`make dev` starts 6 services in parallel but there's no `make stop` to shut them down. An agent must know to use `kill` or `pkill` to stop services, or rely on Ctrl+C in the terminal. There's no `make restart`, no `make status`, no way to check which services are running.

For an agent, service management is important: if a service crashes or gets into a bad state, the agent needs a reliable way to restart it.

**Improvement strategy:** Add service management targets. The simplest approach: `make stop` that kills air and vite processes. A more robust approach: a process manager (like `overmind` or `foreman`) that manages the process group. The right level of investment depends on how often agents encounter service management issues.

### 6. No filtered unit test running (Medium Impact)

Unit tests run all-or-nothing: `make test-unit` runs every unit test in the module. There is no `make test-unit-single TEST=<name>` or pattern-based filtering for unit tests (like the integration test equivalent).

An agent debugging a failing unit test must re-run the entire suite to check if its fix worked. For modules with many tests, this is slow.

**Improvement strategy:** Add a `test-unit-single` target:

```makefile
test-unit-single:
    @if [ -z "$(TEST)" ]; then \
        echo "Usage: make test-unit-single TEST=TestValidateCreateUser"; \
        exit 1; \
    fi
    go test `go list ./... | grep -v test` -run "$(TEST)" -json -v 2>&1 | gotestfmt -hide=empty-packages
```

### 7. No database inspection utilities (Medium Impact)

Beyond `make migrate`, `make seed`, and `make initdb`, there are no database tools. An agent cannot easily:
- View the current schema
- Check what data exists in a table
- Run an ad-hoc query
- Verify that a migration applied correctly

Agents must use raw `psql` commands, which requires knowing the connection string, the table names, and SQL syntax.

**Improvement strategy:** Two approaches:

1. **PostgreSQL MCP server** (from [#17](17-agent-execution-environment.md)) — gives agents structured database access through the MCP protocol
2. **Utility Makefile targets** — `make db-schema` (dump schema), `make db-tables` (list tables), `make db-query SQL="SELECT..."` — lightweight wrappers around psql

The MCP server is the better long-term solution. Makefile targets are a quick win.

### 8. No test output formatting for agents (Low Impact)

Test output goes through `gotestfmt` which formats for human readability (hides empty packages and successful tests). This is good for humans scanning output but means agents don't see successful tests — they only see failures.

For an agent, seeing all test output (including passes) can be useful for understanding test coverage and verifying that the right tests ran.

**Improvement strategy:** Low priority. An agent can always run `go test` directly without `gotestfmt` if it needs full output. No change needed unless agent workflow friction is observed.

### 9. No zoroark build integration (Low Impact)

When the zoroark shared component library changes, frontends that consume it need to pick up the changes. `npm run package` must run in zoroark before frontend builds see updates. There is no root-level target that handles this dependency.

**Improvement strategy:** Include zoroark packaging in the root `make generate` or add a `make build-zoroark` target. This ensures agents modifying shared components don't forget to rebuild the library.

## What Would Be Most Useful for Agents

Prioritized by how often an agent would use the tool and how much friction it removes:

| Priority | Tool | Why |
|---|---|---|
| 1 | `make verify` (lint + check + unit tests) | Every task ends with verification — used every session |
| 2 | `make generate` (all code generation) | Used whenever schema or queries change |
| 3 | `make setup` (bootstrap from fresh clone) | Used once per environment, but blocks everything if missing |
| 4 | [playwright-cli](https://github.com/microsoft/playwright-cli) (browser access) | Used for every frontend task — currently impossible |
| 5 | PostgreSQL MCP (database access) | Used for debugging data issues and verifying migrations |
| 6 | `make stop` / service management | Used when services crash or need restart |
| 7 | `make test-unit-single TEST=<name>` | Used when debugging specific test failures |

## Summary

| Finding | Impact | Action |
|---|---|---|
| No root-level `make verify` (lint + check + test) | High | Add aggregate verification target to root Makefile |
| No root-level `make generate` | High | Add aggregate generation target (workstream #7) |
| No `make setup` bootstrap | High | Add setup target (workstreams #2, #13, #17) |
| No browser/screenshot tool for agents | Medium | Install [playwright-cli](https://github.com/microsoft/playwright-cli) for agent browser access |
| No `make stop` or service management | Medium | Add stop/restart targets or process manager |
| No filtered unit test running | Medium | Add `test-unit-single` target |
| No database inspection utilities | Medium | PostgreSQL MCP server or Makefile wrappers |
| No test output formatting for agents | Low | Low priority — agents can use raw `go test` |
| No zoroark build integration | Low | Include in `make generate` |

The existing tooling is solid for developer use — Makefiles are consistent, hot reload works, test infrastructure is Docker-based and reliable. What's missing is the "agent layer" on top: aggregate commands that span the monorepo (`verify`, `generate`, `setup`), capability-expanding MCP servers (browser, database), and service lifecycle management. The highest-leverage additions are the root-level aggregate targets — they let agents run one command instead of remembering six.
