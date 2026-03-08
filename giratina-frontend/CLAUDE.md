# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in giratina-frontend, a frontend written in TypeScript with SvelteKit and Svelte 5.

## Essential Commands

```bash
npm run check         # Fast static checks: typecheck
npm run lint          # ESLint and Prettier
npm run build         # Production build
```

## Architecture
- `routes/`: SvelteKit file-based routing
- `components/`: Reusable UI components
- `lib/`: Utilities (e.g. fetch wrapper, error handling, routing)

## Testing Strategy
1. **E2E Tests**: Test full user flows with Playwright (Docker)

## Code Patterns and Conventions

### Import conventions

Every file has one canonical import path. Relative imports, `$lib/`, and zoroark barrel imports are banned (enforced by ESLint).

```typescript
// Source files: use $giratina/ prefix
import { createLoadClient, createMutationClient } from "$giratina/lib/api";
import Layout from "$giratina/components/Layout.svelte";

// Zoroark: use deep imports (one per component/utility)
import Button from "@dislyze/zoroark/Button";
import { toast } from "@dislyze/zoroark/toast";
import { type Me } from "@dislyze/zoroark/meCache";

// Exception: ./$types is SvelteKit magic — cannot be aliased
import type { PageData } from "./$types";
```

### Frontend API Calls

Typed API clients generated from OpenAPI spec (`src/schema.ts`). Types are auto-generated — never hand-edit `schema.ts`.

```typescript
// For load functions (SvelteKit load) — must pass SvelteKit's fetch
const api = createLoadClient(fetch);
const { data } = await api.GET("/tenants");
// data! is safe because middleware throws on all errors before returning

// For mutations (Svelte components) — no fetch needed
const api = createMutationClient();
const { data, error } = await api.POST("/tenants/{id}/update", {
  params: { path: { id: tenantId } },
  body: { name: "...", enterprise_features: ... }
});
if (!error) { /* success */ }
```

`$giratina/lib/fetch` exports `handleLoadError` for `{:catch}` blocks in page components. `+layout.ts` has its own local `loadFunctionFetch` for the complex auth logic.

### Frontend Format
Follow the format specified in @giratina-frontend/.prettierrc