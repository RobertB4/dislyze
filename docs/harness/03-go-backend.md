# Workstream #3 — Go Backend Scaffold

## Current Setup

Two Go backends sharing a Go library, all in a Go workspace:
- `lugia-backend/` — customer-facing API (module `lugia`)
- `giratina-backend/` — internal admin API (module `giratina`)
- `jirachi/` — shared library (module `dislyze/jirachi`)

See [#1 repo scaffolding](01-repo-scaffolding.md) for naming discussion.

## What's Already Agent-Friendly

### 1. Handler struct pattern is extremely consistent

Every feature domain follows the same pattern:
```go
type XxxHandler struct {
    dbConn  *pgxpool.Pool
    env     *config.Env
    queries *queries.Queries
}
```
An agent can look at any `handler.go` and immediately replicate the pattern for a new domain.

### 2. One file per endpoint with public/private split

Every endpoint follows the same two-method pattern:
- Public: `func (h *Handler) Login(w, r)` — HTTP layer (decode, validate, call private, respond)
- Private: `func (h *Handler) login(ctx, req)` — business logic (DB queries, rules, transactions)

This is highly replicable. An agent reads one endpoint file and knows the shape of all of them.

### 3. Request validation is mechanical

Every request struct has a `Validate()` method that trims whitespace first, then checks required fields. Always returns `fmt.Errorf("field is required")` style errors. Simple, predictable.

### 4. Transaction pattern is consistent

Every write operation follows the same rollback pattern:
```go
tx, err := h.dbConn.Begin(ctx)
defer func() {
    if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
        errlib.LogError(...)
    }
}()
qtx := h.queries.WithTx(tx)
// ... work ...
tx.Commit(ctx)
```
Note: the rollback checks both `pgx.ErrTxClosed` AND `sql.ErrTxDone`. An agent writing only one check would generate spurious error logs.

### 5. Error handling is uniform

`errlib.New(err, statusCode, userMessage)` → `responder.RespondWithError(w, appErr)`. Same everywhere.

### 6. Middleware layering is clear and documented in CLAUDE.md

The order is strict and documented:
```
Authenticate → LoadTenantAndUserContext → IPWhitelist → RequireFeature → RequirePermission → Handler
```

### 7. SQLC is exemplary

SQL-first, type-safe, generated code never edited directly. The workflow is completely mechanical: write SQL → `make sqlc` → use generated Go.

## What's NOT Agent-Friendly

### 1. Enterprise feature flag system requires changes in 4 files with string coupling (High Impact)

Adding a new enterprise feature requires coordinated changes across:
1. `jirachi/authz/enterprise_features.go` — struct field + sub-type
2. `jirachi/ctx/ctx.go` — `switch` case in `GetEnterpriseFeatureEnabled` (raw string match)
3. `lugia/lib/authz/enterprise_features.go` — `const` + `switch` case in `TenantHasFeature`
4. `lugia/lib/middleware/enterprise_features.go` — `RequireXxx()` helper

The string coupling between layers 2 and 3 is the danger: if the strings diverge, the feature check silently returns `false`. No compiler error, no runtime error, just broken behavior.

This IS documented in lugia's CLAUDE.md with a step-by-step example, which helps. But the coupling is implicit in the code.

**Improvement strategy:**
- Consider generating the switch/case mappings from a single source of truth
- Or add a structural test that validates all `EnterpriseFeature` constants have corresponding `switch` cases in both files
- At minimum, add a code comment in `ctx.go` that says "if you add a case here, also add one in lugia/lib/authz"

### 2. Context getters panic on missing values (High Impact)

All `libctx.GetXxx()` functions use direct type assertions:
```go
func GetTenantID(ctx context.Context) pgtype.UUID {
    tenantID := ctx.Value(TenantIDKey).(pgtype.UUID)
    return tenantID
}
```
If called outside the authenticated middleware chain, this panics. There are no safe getter variants.

An agent adding a new endpoint and accidentally placing it outside the auth middleware group will introduce a panic-on-nil bug that only manifests at runtime.

**Improvement strategy:**
- Add safe getter variants that return `(value, bool)` or `(value, error)`
- Or add a structural test that validates all routes using `libctx.GetXxx()` are inside the auth middleware group
- Or document this loudly in CLAUDE.md: "NEVER call libctx.GetXxx outside authenticated routes"

### 3. POST-for-all-mutations convention is non-standard (Medium Impact)

All mutations use `POST`, even updates and deletes: `POST /{roleID}/update`, `POST /{userID}/delete`. This is a deliberate choice but deviates from REST conventions that agents are trained on. An agent will default to `PUT` for updates and `DELETE` for deletes.

**Improvement strategy:** Document this in CLAUDE.md explicitly: "All mutations use POST. Never use PUT, PATCH, or DELETE methods."

### 4. Japanese user-facing errors vs. empty strings (Medium Impact)

`errlib.New(err, statusCode, userMessage)` has a subtle behavior:
- `userMessage = ""` → response has status code but NO JSON body
- `userMessage = "日本語メッセージ"` → response has `{"error": "日本語メッセージ"}`

An agent must know which errors are user-facing (need Japanese message with domain knowledge) vs. internal (empty string). Getting this wrong either leaks internal errors to users or removes helpful user messages.

**Improvement strategy:** Document the decision criteria in CLAUDE.md: "Use a Japanese error message only when the user needs server-side knowledge to understand what happened (e.g., 'this email is already in use'). For all other errors, pass empty string."

### 5. Two backends look similar but have meaningfully different scopes (Medium Impact)

Lugia and giratina share the same handler struct pattern, but differ significantly in scope. An agent that learns from lugia and applies the same patterns to giratina will over-engineer. An agent that learns from giratina and applies to lugia will under-engineer.

**Improvement strategy:** Each backend's CLAUDE.md should describe its fundamental role and scope abstractly, so agents can derive the right behavior for any scenario — not just list specific differences.

For example:
- "Lugia is a tenant-scoped, customer-facing API. Every request operates within a single tenant's context, with enterprise features, RBAC, and rate limiting applied per-tenant."
- "Giratina is a cross-tenant admin API for internal operators with full access. It has no per-tenant middleware because admins operate across all tenants, not within one."

From this, an agent can derive the right answer for any future question (new enterprise feature → lugia, new bulk admin operation → giratina) rather than checking a list that may be outdated.

### 6. Two sets of SQLC-generated queries (Medium Impact)

`jirachi/queries/` and `lugia-backend/queries/` are separate SQLC generations from the same database schema. Jirachi generates queries for auth middleware tables (refresh tokens, users, tenants). Lugia generates queries for all application tables.

An agent might try to use jirachi's query functions from lugia business logic, or vice versa. Each package has its own `Queries` struct — they're not interchangeable.

**Improvement strategy:** Document in CLAUDE.md: "jirachi/queries is ONLY for the auth middleware. Application code uses lugia-backend/queries (or giratina-backend/queries). Never cross-import."

### 7. Implicit conventions not captured anywhere (Medium Impact)

Several conventions exist only in the code, not in any documentation:
- `dbConn` is for transactions, `queries` is for reads
- Mutation endpoints return 200 with no body (clients re-fetch via GET)
- `is_internal_admin` vs `is_internal_user` are separate flags with different semantics
- Soft delete via `deleted_at IS NULL` is not enforced by the DB

**Improvement strategy:** Add these to lugia's CLAUDE.md as explicit conventions.

### 8. Minor inconsistencies (Low Impact)

- **Test file naming:** `login_test.go` vs `accept_invite_unit_test.go` — inconsistent `_unit_` infix
- **Action constants** are plain strings while Resource is a typed string — inconsistent type safety
- **Two logging systems:** `errlib.LogError` uses stdlib `log.Printf`, auth events use structured JSON logger
- **Named vs positional SQL params:** both `$1` and `@param_name` used in the same file
- **Time serialization:** lugia returns pgx types (auto-marshaled), giratina manually formats to RFC3339
- **Giratina stub route:** `r.Get("/users", ...)` returns hardcoded JSON — dead code

**Improvement strategy:** Low priority. Fix incrementally as we touch these areas. The inconsistencies are minor and unlikely to cause agent errors.

## Summary

| Finding | Impact | Action |
|---|---|---|
| Feature flags require 4-file changes with string coupling | High | Add structural test or generate from single source |
| Context getters panic on missing values | High | Add safe variants or structural test for route placement |
| POST-for-all-mutations not documented | Medium | Add to CLAUDE.md |
| Japanese error message vs empty string decision criteria | Medium | Add decision criteria to CLAUDE.md |
| Two backends look similar but differ meaningfully | Medium | Clarify differences in each CLAUDE.md |
| Two sets of SQLC queries can be confused | Medium | Document cross-import prohibition |
| Implicit conventions not captured | Medium | Add to CLAUDE.md |
| Minor inconsistencies (naming, logging, types) | Low | Fix incrementally |

The Go backend is already well-structured for agents. The highest-leverage improvements are **documenting implicit conventions** (most gaps are knowledge gaps, not structural gaps) and **adding a structural test for the enterprise feature flag system** (the one place where agents are most likely to make silent errors).
