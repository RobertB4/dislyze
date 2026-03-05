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
import { mutationFetch } from "$lugia/lib/fetch";
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
```typescript
// For load functions (GET)
const data = await loadFunctionFetch<Type>('/api/endpoint');

// For mutations (POST/PUT/DELETE)
const {response, success} = await mutationFetch('/api/endpoint', {
  method: 'POST',
  body: JSON.stringify(data)
});
```

### Frontend Format
Follow the format specified in @lugia-frontend/.prettierrc