# Workstream #4 — Database Layer

## Current Setup

- **Schema**: Single migration file (`database/migrations/1_initial_schema.sql`) containing all tables, indexes, triggers, and seed permission data
- **Migrations**: goose v3
- **Code generation**: SQLC with three independent configs (lugia, giratina, jirachi), all pointing to the same shared schema
- **SQL source**: `queries_pregeneration/` directories in each Go module
- **Utility scripts**: `database/seed.sql`, `database/drop.sql`, `database/delete.sql`

## What's Already Agent-Friendly

### 1. SQLC workflow is completely mechanical

Write SQL → `make sqlc` → get typed Go. No ambiguity, no manual editing of generated code. An agent can add a query by writing SQL and running one command.

### 2. Consistent table naming conventions

Tables follow clear patterns:
- `snake_case` plural nouns
- Token tables: `{noun}_tokens`
- Junction tables: `{left}_{right}`
- Indexes: `idx_{table}_{columns}`
- Constraints: `uq_{table}_{description}`
- All PKs: `UUID DEFAULT uuid_generate_v4()`
- All timestamps: `TIMESTAMP WITH TIME ZONE`

### 3. Token tables follow a uniform pattern

Every token table has: `token_hash`, `user_id`/`tenant_id`, `expires_at`, `used_at`, `created_at`. The hash-not-plaintext convention is consistent. An agent can copy any token table as a template.

### 4. Seed data uses readable UUIDs

`seed.sql` uses fixed, pattern-based UUIDs (`11111111-...`, `22222222-...`) making test data easy to reference and reason about.

### 5. `delete.sql` and `drop.sql` handle dependency ordering

Both scripts list tables in reverse FK order, so they work without constraint violations.

## What's NOT Agent-Friendly

### 1. Single migration file means no pattern for incremental migrations (High Impact)

Everything is in `1_initial_schema.sql`. An agent doesn't have an example of what a second migration looks like. Adding a column? New table? Index? There's no reference migration to copy.

**Improvement strategy:** Create at least one incremental migration (even a small one) as a reference. Document the pattern in CLAUDE.md:
```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN foo VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS foo;
-- +goose StatementEnd
```

### 2. Jirachi's queries are stale and diverge from the schema (High Impact)

Jirachi's generated models are missing columns that now exist on the schema (`auth_method` on tenants, `is_internal_user`/`deleted_at`/`external_sso_id` on users). More critically:

- `jirachi/CreateTenant` only inserts `name`, but `auth_method` is `NOT NULL` with no default — **this query will fail at runtime**
- `jirachi/GetUserByID` and `ExistsUserWithEmail` have no `deleted_at IS NULL` filter — they'll find soft-deleted users
- Jirachi is on SQLC v1.28.0 while lugia/giratina are on v1.29.0

An agent running `make sqlc` in jirachi will get unexpected model changes. An agent writing queries against users in jirachi won't know about the soft-delete filter convention.

**Improvement strategy:**
- Update jirachi's SQLC to v1.29.0
- Re-run `make sqlc` and fix any resulting code changes
- Add `deleted_at IS NULL` filters to jirachi's user queries
- Fix `CreateTenant` to include `auth_method` (or evaluate if jirachi should even have `CreateTenant`)
- Add a CI check that validates all three modules generate cleanly against the current schema

### 3. Duplicated queries across modules with semantic divergence (Medium Impact)

`GetUserByID`, `GetTenantByID`, `CreateRefreshToken`, etc. exist in multiple modules. Most are identical SQL, but some diverge:
- Lugia/giratina add `deleted_at IS NULL`; jirachi doesn't
- `CreateUser` has 7 params in lugia, 5 in jirachi
- `CreateTenant` has 3 params in lugia, 1 in jirachi

An agent might assume a query works the same way across modules when it doesn't.

**Improvement strategy:** Document in CLAUDE.md which module owns which queries and why they differ. The abstract principle: "jirachi provides minimal auth-only queries for the shared middleware. Application-level queries with full business logic (soft deletes, feature flags, RBAC) belong in the application module."

### 4. Magic strings embedded in SQL queries (Medium Impact)

Several queries hardcode Japanese role names:
- `'閲覧者'` (viewer) as the RBAC fallback role name
- `'管理者'` (admin) appears in seed data

If these names change in the schema or seed data, the queries silently break. There's no constant or reference — it's a raw string in SQL.

**Improvement strategy:** Add a comment at the top of the relevant SQL files explaining these magic strings and where they come from. Consider whether a DB-level mechanism (e.g., a `role_type` column) would be more robust than matching by name.

### 5. `permissions` table is hardcoded in the migration with fixed UUIDs (Medium Impact)

The 8 permission rows (with Japanese descriptions and fixed UUIDs) are `INSERT`ed in the migration file itself. Adding a new permission requires:
1. A new migration with an `INSERT` using a new fixed UUID
2. Updating seed data to assign the permission to relevant roles
3. Adding a Go constant in `lib/authz/permissions.go`

There's no automated way to keep these in sync. An agent could add the Go constant but forget the migration INSERT (or vice versa).

**Improvement strategy:** Add a structural test that validates all permission constants in Go code have corresponding rows in the database. Or document the multi-step process explicitly.

### 6. `users.email` is globally unique, not per-tenant (Low Impact — but a gotcha)

The `UNIQUE(email)` constraint means the same email cannot exist across tenants. An agent writing test fixtures or seed data with duplicate emails across tenants will get a constraint violation with no obvious explanation.

**Improvement strategy:** Document this in the database CLAUDE.md: "Email uniqueness is global, not per-tenant. This is intentional for SSO domain mapping."

### 7. `queries_pregeneration/` naming (previously flagged)

See [#1 repo scaffolding](01-repo-scaffolding.md#4-queries_pregeneration-is-a-non-standard-name-medium-impact).

### 8. Inconsistent SQL style (Low Impact)

- `CURRENT_TIMESTAMP` vs `NOW()` — both appear, functionally equivalent but inconsistent
- Named params (`@param`) vs positional (`$1`) — both used, roughly split by query complexity
- `RETURNING *` vs `RETURNING specific_columns` — no clear rule

**Improvement strategy:** Low priority. Could standardize on one style per convention (e.g., "always use `@named_params` for queries with 3+ parameters") but the inconsistency is unlikely to cause agent errors.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No incremental migration example | High | Create a reference migration, document the pattern |
| Jirachi queries are stale / diverge from schema | High | Update SQLC, fix queries, add CI generation check |
| Duplicated queries with semantic differences | Medium | Document which module owns which queries and why |
| Magic Japanese strings in SQL | Medium | Comment the strings, consider a `role_type` column |
| Hardcoded permissions in migration | Medium | Add structural test or document the multi-step process |
| Global email uniqueness is non-obvious | Low | Document in database CLAUDE.md |
| Inconsistent SQL style | Low | Standardize incrementally |

The SQLC workflow is the strongest part of the database layer — it's exactly what a harness wants (mechanical, type-safe, generated). The main risks are **stale jirachi queries** (a live bug waiting to happen) and the **lack of incremental migration examples** (agents won't know the pattern for schema changes).
