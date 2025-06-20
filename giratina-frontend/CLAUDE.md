# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in giratina-frontend, a frontend written in TypeScript with SvelteKit and Svelte 5.

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
Follow the format specified in @giratina-frontend/.prettierrc