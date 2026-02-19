# Workstream #5 — OpenAPI Contract Layer

## Current Setup

There is no OpenAPI spec. The contract between frontend and backend is entirely **manual and trust-based**.

### How it works today

**Backend (Go):** Handlers write JSON responses by constructing anonymous structs or using SQLC-generated models directly. There is no formal definition of what each endpoint accepts or returns.

**Frontend (TypeScript):** Types are hand-written and co-located with the fetch calls that use them, inside `+page.ts` files. The frontend casts `response.json()` with `as Type` — a compile-time assertion with zero runtime validation.

**Shared types:** The `Me` and `EnterpriseFeatures` types live in zoroark (shared UI library). Everything else is defined per-route.

### The fetch layer

Two utilities exist (identically duplicated in both frontends):
- `loadFunctionFetch(fetch, url)` → returns raw `Response` (for SvelteKit load functions)
- `mutationFetch(url, options)` → returns `{ response: Response; success: boolean }` (for component mutations)

Neither function is typed to the endpoint being called. The caller must know the URL, HTTP method, request body shape, and response shape — all from memory or by reading backend code.

### Type duplication

Some types are defined in multiple places:
- `Permission` and `RoleInfo` are defined identically in both `settings/users/+page.ts` and `settings/roles/+page.ts`
- Giratina extends the shared `EnterpriseFeatures` type with SSO fields, creating a second variant

## What's Already Agent-Friendly

### 1. Types are co-located with their usage

Following "locality of behavior," each `+page.ts` defines the types it needs right next to the fetch call. An agent working on a specific page can see the full picture in one file.

### 2. The `Me` type is centralized

The most important shared type (`Me`) lives in zoroark and is used consistently by both frontends. An agent knows to look there for the current user shape.

### 3. Fetch utilities handle auth/error uniformly

`loadFunctionFetch` and `mutationFetch` handle 401 redirects, error toasts, and network failures consistently. An agent making API calls just uses these wrappers.

## What's NOT Agent-Friendly

### 1. No single source of truth for the API contract (High Impact)

When an agent changes a Go handler's response shape (e.g., adds a field, renames a field, changes a type), nothing tells it to update the frontend. The TypeScript types are disconnected from the Go code. The change will silently break at runtime — no compiler error, no test failure (unless e2e tests happen to cover that field).

This is the fundamental gap that an OpenAPI spec would solve. The spec becomes the single source of truth, and both sides are generated or validated from it.

### 2. Frontend types are untyped `as` casts with no runtime validation (High Impact)

```typescript
const data = await response.json() as GetUsersResponse;
```

This tells TypeScript to trust the shape, but if the backend returns something different, there's no error — just undefined fields or wrong types at runtime. An agent writing new fetch calls will follow this pattern and create the same fragile coupling.

With an OpenAPI-generated client, the types would be generated from the spec and the fetch calls would be type-safe by construction.

### 3. Duplicated type definitions across routes (Medium Impact)

`Permission`, `RoleInfo`, `PaginationMetadata` are defined multiple times in different `+page.ts` files. An agent changing the permission structure would need to find and update every copy. With a generated client, these types exist once.

### 4. No documentation of what each endpoint accepts/returns (Medium Impact)

An agent implementing a new frontend page that calls an existing endpoint must read the Go handler code to understand:
- What URL to call
- What HTTP method to use (always POST for mutations, but which path?)
- What request body shape is expected
- What response body shape comes back
- What error codes are possible

An OpenAPI spec would answer all of these questions from a single file.

### 5. `loadFunctionFetch` returns raw `Response` — no per-endpoint typing (Medium Impact)

The fetch wrapper returns `Response`, not a typed result. Every caller must manually cast. With a generated client, you'd call `api.users.list()` and get `Promise<GetUsersResponse>` directly.

## The Gap: What Would an OpenAPI Layer Look Like?

### The type safety chain we discussed earlier

```
SQL schema
  → SQLC generates Go types + query functions
    → Go handler uses those types
      → Go handler implements OpenAPI-generated interface  ← NEW
        → OpenAPI spec is the contract                     ← NEW
          → openapi-typescript generates TS types + client  ← NEW
            → Svelte components use generated client
```

### Approach options

**Option A: Spec-first** — Write the OpenAPI spec by hand, generate both Go server interfaces and TypeScript client from it.
- Pro: Spec is the single source of truth
- Con: Maintaining the spec manually is another thing to keep in sync

**Option B: Code-first** — Generate the OpenAPI spec from Go code (using annotations or reflection), then generate the TypeScript client from the spec.
- Pro: Go code is the source of truth (no extra file to maintain)
- Con: Needs annotation tooling; generated spec may be noisy

**Option C: Lightweight contract testing** — Don't introduce OpenAPI. Instead, add contract tests that validate the frontend's expected types against the backend's actual responses.
- Pro: Least disruptive to current codebase
- Con: Doesn't give you a generated client or typed fetch calls

### What changes in the codebase

For Option A or B, the impact would be:
1. New `api/` directory at root for the OpenAPI spec (or generated spec output)
2. Generated Go server interfaces that handlers must implement
3. Generated TypeScript client replacing manual fetch calls + `as Type` casts
4. `make generate` target updated to include OpenAPI generation
5. CI gate that validates spec → generated code → types are all in sync
6. Removal of hand-written types from `+page.ts` files (replaced by imports from generated client)

### Recommendation

For a harness engineering environment, **Option A (spec-first)** is the strongest choice because:
- The spec is a readable, diffable, reviewable document
- Agents can read the spec to understand the full API without reading Go code
- Both sides are mechanically constrained by the same source of truth
- Adding a new endpoint follows a mechanical workflow: update spec → generate → implement

However, this is a significant investment. A phased approach could work:
1. **Phase 1:** Generate an OpenAPI spec from the existing endpoints (snapshot the current state)
2. **Phase 2:** Generate the TypeScript client from the spec, replace manual types
3. **Phase 3:** Generate Go server interfaces, enforce handler conformance
4. **Phase 4:** CI gate that rejects drift between spec, server, and client

## Summary

| Finding | Impact | Action |
|---|---|---|
| No single source of truth for API contract | High | Introduce OpenAPI spec |
| Frontend types are unvalidated `as` casts | High | Replace with generated client types |
| Duplicated type definitions across routes | Medium | Generated client eliminates duplication |
| No endpoint documentation | Medium | OpenAPI spec serves as living documentation |
| Fetch wrapper returns untyped `Response` | Medium | Generated client provides typed methods |

This workstream is the biggest structural gap in the current codebase for agent-first development. Currently, an agent changing a backend response has no mechanical way to know the frontend will break. An OpenAPI contract layer closes this gap entirely — it turns a silent runtime failure into a compile-time error.
