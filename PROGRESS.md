# Progress: End-to-End Type Safety with Huma

## Branch: `introduce-openapi`

## Completed

### Backend: Huma migration
- All lugia-backend handlers migrated from Chi-style to huma
- All giratina-backend handlers migrated from Chi-style to huma (including LogInToTenant redirect via `Location` header)
- `lib/humautil/humautil.go` in both backends:
  - `NewError(err, status)` — logs internally, returns generic error to client
  - `NewErrorWithDetail(err, status, detail)` — logs internally, returns specific user-facing message
  - `huma.NewError` override logs huma's validation details server-side
- `lib/middleware/inject_raw_http.go` in both backends — stores `*http.Request` / `http.ResponseWriter` in context for handlers needing cookie/rate-limit access
- `cmd/openapi/main.go` in both backends — offline OpenAPI spec generation
- OpenAPI specs committed as `openapi.json`

### Frontend: Typed API clients
- Both frontends use `openapi-fetch` with generated TypeScript schemas (`src/schema.ts`)
- `createLoadClient(fetch)` for SvelteKit load functions (throws on errors, `data!` is safe)
- `createMutationClient()` for mutations (toast on error, check `!error` for success)
- `+layout.ts` kept as-is in both frontends (complex auth logic uses legacy `loadFunctionFetch`)
- `handleLoadError` from legacy fetch still used in `{:catch}` blocks

### Error handling architecture
- Every error is logged server-side (restored guarantee from old `responder.RespondWithError`)
- `huma.NewError` override ensures all errors (including huma's own validation) use `{"error": "message"}` format matching frontend expectation
- Huma validation errors logged via `log.Printf` in the override for debugging frontend/backend validation mismatches

## Follow-up items (separate PRs)

### ~~Inline legacy fetch and update CLAUDE.md~~ ✅
- Moved `loadFunctionFetch` into `+layout.ts` as a local function in both frontends (only consumer)
- Removed dead `mutationFetch` from both `fetch.ts` files (no callers — replaced by `createMutationClient`)
- `fetch.ts` now only exports `handleLoadError` (still used by page components for `{:catch}` blocks)
- Updated CLAUDE.md in both frontends: removed legacy fetch references, documented `createLoadClient`/`createMutationClient` as the only API patterns
- Updated JSDoc comments in both `api.ts` files to remove legacy references

### CI for code generation
- Add a CI step that runs `make generate` (SQLC) and `go run ./cmd/openapi` (OpenAPI specs) and fails if the output differs from what's committed
- Catches cases where someone changes types/queries but forgets to regenerate

### ~~Consolidate error handling: humautil → jirachi/errlib~~ ✅
- Removed `errlib.New` / `errlib.AppError` — replaced with `errlib.NewError` / `errlib.NewErrorWithDetail`
- `APIError` in `jirachi/errlib` satisfies `huma.StatusError` via structural typing (no huma import)
- `newHumaConfig` kept as unexported function in each backend's `main.go`
- Deleted `lib/humautil/` from both backends
- Simplified all handler boilerplate (removed AppError unwrapping, handlers just `return nil, err`)
- End state: `jirachi/errlib` is the single place for all error types and constructors

### ~~Rename StoreHTTPRequest middleware~~ ✅
- Renamed `StoreHTTPRequest` → `InjectRawHTTP` in both backends
- Renamed file `store_request.go` → `inject_raw_http.go`
- Updated all references in `main.go` of both backends

### SSO endpoints — not migrating to huma
- `SSOLogin`, `SSOACS`, `SSOMetadata` in lugia-backend remain Chi-style handlers
- Reason: they handle non-JSON payloads (SAML form-encoded POST, XML metadata) that don't fit huma's JSON-centric model
- This is intentional, not tech debt

### ~~SkipValidateBody workarounds~~ ✅
- Added `omitempty` to `company_name` and `user_name` in `GenerateTenantInvitationTokenRequest` (request-only type)
- Added `omitempty` to all fields in `authz.EnterpriseFeatures` and sub-types (`RBAC`, `IPWhitelist`, `SSO`)
- Removed `SkipValidateBody: true` from both `GenerateTokenOp` and `UpdateTenantOp`
- Regenerated OpenAPI specs and frontend TypeScript schemas
- Fixed TypeScript non-null assertions in giratina-frontend `+page.svelte` for optional feature access

### ~~Unify validation on huma~~ ✅
- Replaced all `Validate()` methods with huma struct tags (`minLength`, `maxLength`, `minItems`, `pattern`) and `Resolve()` (huma Resolver interface) for cross-field validation
- **Simple cases** (tags only): login, signup fields, create/update role, update user roles, verify reset token, update me, change tenant name, forgot password email, change email, invite user, update IP label, add IP to whitelist
- **Complex cases** (tags + Resolver): reset password, accept invite, signup, change password (password match), add IP to whitelist (CIDR validation), generate tenant invitation token (conditional SSO validation)
- Removed 12 obsolete `Validate()` unit test files — validation now covered by huma framework and integration tests
- Only remaining `Validate()`: `sso_login.go` (Chi-style handler, not huma — intentionally kept)
- Regenerated OpenAPI specs and frontend TypeScript schemas
- Validation errors now return 422 (huma default) instead of 400 for schema/tag violations

### Fix nullable slices in OpenAPI spec
- Go nil slices serialize to `null` in JSON, so the OpenAPI spec generates `T[] | null` for slice fields
- This forces frontend code to use `?? []` defensively (e.g. `data!.tenants ?? []`)
- Fix: ensure Go handlers always return initialized slices (not nil), or use appropriate huma tags
