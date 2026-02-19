# Workstream #8 — Testing Infrastructure

## Current Setup

Three test layers exist with varying coverage:

| Layer | Tool | Where | Coverage |
|---|---|---|---|
| Go unit tests | `go test` + testify | `lugia-backend/`, `jirachi/` | Validation functions only |
| Go integration tests | `go test` in Docker | `lugia-backend/test/`, `giratina-backend/test/` | Most API endpoints |
| E2E tests | Playwright | `lugia-frontend/test/` | Customer-facing UI flows |
| Frontend unit tests | — | — | None exist |

All test layers run in CI. Integration and E2E tests use Dockerized environments with a real PostgreSQL database, seed data, and mock services (SendGrid, Keycloak).

### Test data

A shared `database/seed.sql` (381 lines) provides the canonical test fixtures — 5 tenants, ~100 users, roles, permissions, invitation tokens. This data is mirrored as typed constants in:
- Go: `lugia-backend/test/integration/setup/seed.go` (822 lines)
- TypeScript: `lugia-frontend/test/e2e/setup/seed.ts` (539 lines)

Both mirrors are maintained manually — there is no generation from the SQL.

## What's Already Agent-Friendly

### 1. Integration tests are the gold standard

Integration tests make real HTTP calls to the running service, assert against real database state, and cover auth flows end-to-end including email verification (via mock SendGrid with polling/retry). The `security_test.go` (414 lines) tests SQL injection, XSS, privilege escalation, and JWT tampering. An agent can add a new endpoint and write a corresponding integration test by copying any existing test file.

### 2. Every test layer follows the same structure

- Go unit tests: table-driven with `testify/assert`
- Go integration tests: `setup.InitDB` → `setup.ResetAndSeedDB` → HTTP calls → assertions → `defer setup.CloseDB`
- E2E tests: `resetAndSeedDatabase()` in `beforeAll` → `logInAs()` in `beforeEach` → Playwright assertions

An agent reading one test at each layer understands the pattern for all of them.

### 3. Seed data has typed constants

Test fixtures are not magic strings scattered across test files. Both Go and TypeScript exports have typed maps (`TestUsersData.enterprise_1.userID`, `TestTenantsData.enterprise.tenantID`) so an agent can reference test data by name and get autocomplete.

### 4. `data-testid` attributes on all interactive elements

The frontend consistently uses `data-testid` for Playwright locators. An agent writing E2E tests can reliably select elements without fragile CSS selectors.

### 5. `make test-integration-single TEST=FunctionName`

An agent debugging a specific integration test can run just that test inside Docker without waiting for the full suite. This fast feedback loop is critical for iterative test development.

## What's NOT Agent-Friendly

### 1. Go unit tests only cover validation — no handler logic testing (High Impact)

Every Go unit test exclusively tests the `Validate()` method on request body structs. There are no unit tests for handler logic, middleware, context utilities, error handling, or any business logic beyond input validation.

This means any change to handler behavior requires running the full Docker integration test suite to verify correctness. For an agent, this turns a 2-second feedback loop into a 30+ second one. It also means an agent cannot test handler logic in isolation — there's no mocking layer for the database interface.

SQLC generates a `Querier` interface (`emit_interface: true` in all configs), which is designed to be mockable. But no mock implementations exist anywhere in the codebase.

**Improvement strategy:** Create a mock implementation of the SQLC `Querier` interface (either hand-written or using a tool like `mockgen`). Document the pattern in CLAUDE.md: "For new handlers, write unit tests that mock the `Querier` interface. Integration tests verify the real database; unit tests verify the handler logic." This gives agents a fast feedback loop for handler changes.

### 2. Go and TypeScript seed data are manually synced (High Impact)

`seed.go` (822 lines) and `seed.ts` (539 lines) both mirror `database/seed.sql` but are maintained independently. If `seed.sql` adds a new user or changes a UUID, both files must be updated. An agent modifying seed data will likely update one and miss the other — or update the SQL and miss both.

**Improvement strategy:** Consider generating `seed.go` and `seed.ts` from `seed.sql` (or from a shared JSON/YAML fixture definition). Alternatively, add a structural test that validates the constants in both files match the rows in `seed.sql`. At minimum, document in CLAUDE.md: "After changing `database/seed.sql`, update `lugia-backend/test/integration/setup/seed.go` AND `lugia-frontend/test/e2e/setup/seed.ts` to match."

### 3. No frontend unit tests exist (Medium Impact)

There are no Vitest, Jest, or any other frontend unit tests. No test runner is configured. All frontend validation happens at the E2E level, which is slow and requires the full Docker environment.

For an agent building a new Svelte component or modifying form validation logic, there's no way to test in isolation. The feedback loop is: modify code → build Docker → run Playwright → wait for browser automation.

**Improvement strategy:** Add Vitest to the frontend projects. Start with utility function tests (e.g., the validation logic in Felte forms, any data transformation helpers). Document the pattern in CLAUDE.md. This is lower priority than the Go handler testing gap because most frontend logic is declarative (Svelte templates + Felte forms), but utility functions and complex `$derived` expressions benefit from unit tests.

### 4. Giratina has zero test coverage at frontend level (Medium Impact)

`giratina-frontend/` has no tests of any kind — no E2E, no unit, no integration tests specific to the admin UI. `giratina-backend/` has integration tests but no unit tests. An agent working on giratina features has no safety net.

**Improvement strategy:** Add at minimum an E2E smoke test for giratina (login → view tenants → view tenant detail). The admin panel is simpler than the customer-facing UI, so a small number of E2E tests covers the critical paths. Document in CLAUDE.md that giratina needs E2E coverage.

### 5. SSO integration tests are disabled (Medium Impact)

The mock Keycloak service is commented out in `lugia-backend/test/docker-compose.integration.yml`. SSO flows (SAML ACS, SSO login redirect, SP metadata) have no integration test coverage. The only SSO testing happens at the E2E level via Playwright.

An agent modifying SSO handler code cannot verify their changes without running the full E2E suite.

**Improvement strategy:** Uncomment mock-keycloak in the integration Docker Compose and add integration tests for the SSO endpoints. This gives agents a faster feedback loop for SSO changes.

### 6. No test coverage measurement or thresholds (Medium Impact)

No `go test -cover` flag, no coverage reports, no coverage thresholds in CI. An agent has no way to know if their new code is covered by existing tests, and there's no gate preventing coverage regression.

**Improvement strategy:** Add `-cover -coverprofile=coverage.out` to the Go test commands. Add a CI step that reports coverage (even without a hard threshold). For the frontend, add coverage reporting when Vitest is introduced. Coverage thresholds can be added later — the first step is visibility.

### 7. Jirachi has almost no tests (Low Impact)

Only `jirachi/jwt/jwt_test.go` exists. The shared library's middleware, context utilities, error handling, rate limiting, logging, and SendGrid integration have no tests. Changes to jirachi affect both backends but are only caught by downstream integration tests.

**Improvement strategy:** Since jirachi is a shared library, its functions should be the most well-tested code in the repository. Add unit tests for at least: `ctx.GetXxx()` functions (document the panic behavior), `errlib.New()` / `errlib.LogError()`, `ratelimit` middleware, and `auth.Middleware()` (with mocked dependencies).

### 8. Test file naming is inconsistent (Low Impact)

Some test files use `_unit_test.go` suffix, others use plain `_test.go`. Both are unit tests in the same directory. An agent might be confused about whether `_unit_test.go` has different semantics.

**Improvement strategy:** Standardize on one convention. Since Go's test runner treats all `_test.go` files the same, the `_unit_` infix adds no mechanical value. Either use it everywhere (to distinguish from integration tests visually) or remove it everywhere.

## Summary

| Finding | Impact | Action |
|---|---|---|
| Unit tests only cover validation, not handler logic | High | Create Querier mock, document handler test pattern |
| Go + TS seed data manually synced with seed.sql | High | Generate from single source or add sync validation |
| No frontend unit tests | Medium | Add Vitest, start with utility functions |
| Giratina has zero frontend test coverage | Medium | Add E2E smoke test |
| SSO integration tests disabled | Medium | Uncomment mock-keycloak, add SSO integration tests |
| No test coverage measurement | Medium | Add coverage flags and CI reporting |
| Jirachi (shared lib) has almost no tests | Low | Add unit tests for critical shared functions |
| Inconsistent test file naming | Low | Standardize `_test.go` convention |

The integration test layer is the strongest part of the testing infrastructure — it's comprehensive, well-structured, and covers security scenarios. The biggest gap is the **absence of handler-level unit tests** due to no mock layer for the SQLC `Querier` interface. This forces agents to use the slow Docker-based integration test suite for any handler logic change. Adding a mock layer would give agents a fast feedback loop and is the single highest-leverage testing improvement.
