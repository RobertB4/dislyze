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
- `lib/middleware/store_request.go` in both backends — stores `*http.Request` / `http.ResponseWriter` in context for handlers needing cookie/rate-limit access
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

### Inline legacy fetch and update CLAUDE.md
- `loadFunctionFetch` is only used in `+layout.ts` in each frontend — inline it and remove the file
- `handleLoadError` similarly only used in `{:catch}` blocks — inline or simplify
- Update CLAUDE.md files in both frontends: remove references to `loadFunctionFetch`/`mutationFetch`, document `createLoadClient`/`createMutationClient` as the only API patterns

### CI for code generation
- Add a CI step that runs `make generate` (SQLC) and `go run ./cmd/openapi` (OpenAPI specs) and fails if the output differs from what's committed
- Catches cases where someone changes types/queries but forgets to regenerate

### Consolidate error handling: humautil → jirachi/errlib
- Remove `errlib.New` / `errlib.AppError` (replaced by `humautil.NewError` / `NewErrorWithDetail`)
- Move `APIError`, `NewError`, `NewErrorWithDetail` into `jirachi/errlib` — these don't import huma (Go structural typing satisfies `huma.StatusError` implicitly)
- Keep `NewConfig` in each backend (thin wrapper that wires `huma.NewError` to errlib's `APIError`) — can't move to jirachi because it imports `huma/v2`
- Delete `lib/humautil/` from both backends
- End state: `jirachi/errlib` is the single place for all error handling; each backend has only a thin huma config wiring

### Rename StoreHTTPRequest middleware
- `StoreHTTPRequest` doesn't communicate *where* it stores the request (context)
- Rename to something like `InjectRawHTTP` or `StoreHTTPInContext` for clarity
- Update all references in both backends

### SSO endpoints — not migrating to huma
- `SSOLogin`, `SSOACS`, `SSOMetadata` in lugia-backend remain Chi-style handlers
- Reason: they handle non-JSON payloads (SAML form-encoded POST, XML metadata) that don't fit huma's JSON-centric model
- This is intentional, not tech debt

### Unify validation on huma

### Problem
We use a mix of huma struct tags and custom `Validate()` methods. This split confuses agents — they see a huma codebase and naturally reach for huma tags, but some validation lives in `Validate()`. Two systems for the same concern.

### Decision
Embrace huma validation fully:
- **Huma tags** for the ~90-95% standard cases: `required`, `minimum`, `maximum`, `maxLength`, `default`, etc.
- **Huma Resolvers** for the ~5-10% complex cases: cross-field validation (e.g., passwords must match)
- **Remove all `Validate()` methods** — no custom validation pattern
- Tags handle binding (`query`, `path`, `header`), defaults, and validation in one place
- Agents do the right thing by default without special instructions

### Fix nullable slices in OpenAPI spec
- Go nil slices serialize to `null` in JSON, so the OpenAPI spec generates `T[] | null` for slice fields
- This forces frontend code to use `?? []` defensively (e.g. `data!.tenants ?? []`)
- Fix: ensure Go handlers always return initialized slices (not nil), or use appropriate huma tags
- Address during the validation unification pass since it touches the same struct definitions

### Scope
- lugia-backend: Replace all `Validate()` methods with huma tags + Resolvers where needed
- giratina-backend: Same
- Update CLAUDE.md files to document the convention
- Regenerate OpenAPI specs and frontend TypeScript schemas
