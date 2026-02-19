# Workstream #6 — SvelteKit Frontend Scaffold

## Current Setup

Two SvelteKit frontends sharing a Svelte 5 component library:
- `lugia-frontend/` — customer-facing SaaS UI
- `giratina-frontend/` — internal admin panel
- `zoroark/` — shared UI component library (`@dislyze/zoroark` via `file:` reference)

All use Svelte 5 (runes), Tailwind CSS v4, static adapter (SPA mode, no SSR), and Felte for form handling.

## What's Already Agent-Friendly

### 1. Data flow pattern is completely uniform

Every page follows the exact same structure:
1. `+page.ts`: Create unawaited promises via `loadFunctionFetch`, define response types, return promises
2. `+page.svelte`: Destructure props as `let { data: pageData }: { data: PageData } = $props()`, pass promises to Layout
3. Layout resolves promises, renders skeleton while loading, then renders children with resolved data

An agent can copy any existing page as a template.

### 2. Form handling is fully standardized on Felte

Every form uses `createForm` with the same pattern: `initialValues` → `validate` (trim + check) → `onSubmit` (mutationFetch + invalidate + toast + reset). No variation across the codebase.

### 3. Co-located route components

Route-specific components (Skeleton, modals, sub-components) live in the same directory as the `+page.svelte` that uses them. An agent working on a feature has everything in one directory.

### 4. Types co-located with load functions

Response types live in `+page.ts`, imported into `+page.svelte` via `import type { Xxx } from "./+page"`. An agent doesn't need to hunt for type definitions.

### 5. Consistent Svelte 5 rune usage

`$props()`, `$state()`, `$derived`, `$effect()` are used consistently throughout. No Svelte 4 patterns except where Felte requires stores.

### 6. Shared component library has clear props API

Zoroark components all use `$props()` with well-defined TypeScript types. Button, Input, Alert, Badge, Slideover, etc. are reusable building blocks.

### 7. `data-testid` on interactive elements

Every meaningful element has a `data-testid` attribute for Playwright E2E tests. This is a convention an agent can follow mechanically.

## What's NOT Agent-Friendly

### 1. The promise-resolution-in-Layout pattern is non-standard (High Impact)

Data is fetched in `+page.ts` as unawaited promises, passed to a generic `Layout` component, and resolved there via `{#await}`. This is not how SvelteKit tutorials or documentation show data loading. An agent trained on standard SvelteKit patterns will expect:
- `await` in load functions
- `{#await data}` in page templates
- Or SvelteKit's native streaming with `+page.server.ts`

Instead, all async resolution goes through the Layout's generic promise resolver. This is a powerful pattern but completely unique to this codebase.

**Improvement strategy:** Document this pattern prominently in CLAUDE.md with an explanation of WHY it exists (deferred loading, skeleton support, uniform error handling) and a replication recipe. Without this, an agent will write load functions that await and break the pattern.

### 2. `$derived` inconsistency in zoroark components (Medium Impact)

Two different `$derived` patterns coexist:
- `Badge`, `InteractivePill`: `const x = $derived(() => { ... })` — returns a function, called as `{x()}` in template
- `Button`, `Input`: `const x = $derived(\`...\`)` — returns a value, used as `{x}` in template

An agent generating new components might use either form inconsistently. The function-returning form is technically unnecessary — `$derived` without a wrapper function works for all cases.

**Improvement strategy:** Standardize on the value form (`$derived(expression)`). Fix Badge and InteractivePill.

### 3. Felte stores mixed with Svelte 5 runes (Medium Impact)

Felte returns Svelte 3/4-style writable stores (`$data`, `$errors`, `$isSubmitting`). These require the `$` store prefix in templates. Everything else in the codebase uses Svelte 5 runes. An agent trained on Svelte 5 rune-only code might try to access Felte values without the `$` prefix.

**Improvement strategy:** Document in CLAUDE.md: "Felte uses Svelte stores, not runes. Always access Felte values with `$` prefix in templates: `$data.field`, `$errors.field?.[0]`, `$isSubmitting`." Long-term, consider whether Felte should be replaced with a rune-native form library or plain rune-based validation.

### 4. Navigation is duplicated (mobile + desktop) in Layout.svelte (Medium Impact)

Adding a new nav item requires adding it in two places within the same `Layout.svelte` file — once in the mobile nav and once in the desktop sidebar. An agent might add it in one place and miss the other.

**Improvement strategy:** Extract the nav items into a data structure (array of `{ href, label, permission?, feature? }`) and render both mobile and desktop nav from it. This is a structural fix that eliminates the duplication.

### 5. `forceUpdateMeCache` + `invalidate` must be coordinated (Medium Impact)

After profile mutations, both must be called:
```typescript
forceUpdateMeCache.set(true);
await invalidate((u) => u.pathname === "/api/me");
```
Doing one without the other breaks caching. This coordination is not obvious from reading either file alone.

**Improvement strategy:** Create a helper function (e.g., `refreshMe()` in zoroark) that encapsulates both calls. Then document: "After any mutation that changes user data, call `refreshMe()`."

### 6. Implicit conventions not captured in CLAUDE.md (Medium Impact)

Several important conventions exist only in the code:
- Props always destructured as `let { data: pageData }: { data: PageData } = $props()` (never bare `data`)
- Never `await` in load functions — always return unawaited promises
- `resolve()` from `$app/paths` required for all internal links
- Japanese for all UI text
- `$app/state` (Svelte 5), NOT `$app/stores` (Svelte 4) for page state
- Slideover wraps forms, not the other way around (the `<form>` element is the outer wrapper)

**Improvement strategy:** Add these conventions to lugia-frontend's CLAUDE.md. A "new page recipe" section would be particularly valuable.

### 7. Frontend CLAUDE.md files are thin and near-identical (Low Impact)

Both frontend CLAUDE.md files list commands and basic patterns but miss the deeper conventions (Layout pattern, Felte integration, promise flow, invalidation patterns). Giratina's CLAUDE.md is a copy of lugia's with the name changed.

**Improvement strategy:** Flesh out lugia-frontend's CLAUDE.md with the patterns documented here. Giratina's should describe its scope abstractly (same as the backend audit recommendation): "Giratina is an admin panel for internal operators managing tenants across the system. It has no RBAC, no enterprise feature gating, and simpler navigation."

### 8. `$effect` is easy to misuse and agents will reach for it too readily (High Impact)

`$effect` is the most dangerous Svelte 5 rune. It's convenient and familiar (similar to React's `useEffect`), so agents will default to it whenever they need reactive behavior. But `$effect` frequently leads to architecture debt, subtle bugs, and unnecessary complexity. In almost all cases, `$derived`, event handlers, or restructuring the data flow is the correct solution.

Agents should treat `$effect` as a last resort: always explore alternative solutions first — even if those alternatives require bigger refactors or are harder to implement — and only use `$effect` when there is genuinely no other way.

**Improvement strategy:** Add an explicit rule to CLAUDE.md: "Never use `$effect` without first exhausting all alternatives (`$derived`, event handlers, restructured data flow). `$effect` is only acceptable when no other solution exists. If you think you need `$effect`, explain why alternatives don't work." Consider adding a custom lint rule that flags new `$effect` usage for review.

### 9. `lib/fetch.ts` duplicated in both frontends (Low Impact — previously flagged)

See [#1 repo scaffolding](01-repo-scaffolding.md#3-duplicated-code-between-applications-medium-impact). Should be extracted to zoroark.

## Summary

| Finding | Impact | Action |
|---|---|---|
| Non-standard promise-in-Layout pattern undocumented | High | Document pattern + recipe in CLAUDE.md |
| `$effect` misuse risk — agents will default to it | High | Explicit "last resort" rule in CLAUDE.md, consider lint rule |
| `$derived` inconsistency in zoroark | Medium | Standardize on value form |
| Felte stores mixed with Svelte 5 runes | Medium | Document in CLAUDE.md, consider long-term replacement |
| Nav duplication (mobile + desktop) | Medium | Extract nav items to data structure |
| `forceUpdateMeCache` + `invalidate` coordination | Medium | Create `refreshMe()` helper |
| Implicit conventions not in CLAUDE.md | Medium | Add conventions + new page recipe |
| Thin frontend CLAUDE.md files | Low | Flesh out with patterns |
| `lib/fetch.ts` duplication | Low | Extract to zoroark (flagged in #1) |

The frontend is well-structured and consistent — the same patterns repeat across all pages. The main agent risk is the **non-standard promise-resolution pattern** in the Layout component. This is the single most important thing to document because it deviates from how every SvelteKit tutorial teaches data loading. Once an agent understands this pattern, the rest follows naturally.
