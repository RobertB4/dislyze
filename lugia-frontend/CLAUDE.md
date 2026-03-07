# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in lugia-frontend, a frontend written in TypeScript with SvelteKit and Svelte 5.

## Essential Commands

```bash
npm run build         # Production build
npm run test-e2e      # E2E tests (Playwright with Docker)
npm run lint          # ESLint and Prettier
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
// Source files: use $lugia/ prefix
import { createMutationClient } from "$lugia/lib/api";
import Layout from "$lugia/components/Layout.svelte";

// Test files: use $lugia-test/ prefix
import { resetAndSeedDatabase } from "$lugia-test/e2e/setup/helpers";

// Zoroark: use deep imports (one per component/utility)
import Button from "@dislyze/zoroark/Button";
import { toast } from "@dislyze/zoroark/toast";
import { type Me } from "@dislyze/zoroark/meCache";

// Exception: ./$types is SvelteKit magic — cannot be aliased
import type { PageData } from "./$types";
```

### Frontend API Calls

**For load functions:**
Use the typed `openapi-fetch` client. Types are auto-inferred from the URL — no manual type annotations.
```typescript
import { createLoadClient } from "$lugia/lib/api";

export function load({ fetch }: Parameters<PageLoad>[0]) {
  const api = createLoadClient(fetch);
  const usersPromise = api.GET("/users", {
    params: { query: { page: 1, limit: 50 } }
  }).then(({ data }) => data!);
  return { usersPromise };
}
```
`data!` is safe: middleware in `createLoadClient` throws on all error statuses before openapi-fetch returns. See `src/lib/api.ts` for details.

**For mutations:**
```typescript
import { createMutationClient } from "$lugia/lib/api";

const api = createMutationClient();
const { data, error, response } = await api.POST("/users/invite", {
  body: { email: "...", name: "...", role_ids: ["..."] }
});
if (!error) {
  // success — error handling (toast, 401 redirect) is in middleware
}
```

**For types — import directly from schema:**
```typescript
import type { UserInfo, GetUsersResponse } from "$lugia/schema";
```

`$lugia/lib/fetch` exports `handleLoadError` for `{:catch}` blocks in page components. `+layout.ts` has its own local `loadFunctionFetch` for the complex auth logic.

### Frontend Format
Follow the format specified in @lugia-frontend/.prettierrc