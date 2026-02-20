# zoroark — Shared Svelte Component Library

Zoroark (`@dislyze/zoroark`) is the shared UI component library used by both `lugia-frontend` and `giratina-frontend`. Changes here affect both frontends.

## Essential commands

```bash
npm run build     # Build the library (vite build + package + CSS)
npm run package   # Package components for consumption
npm run check     # TypeScript/Svelte type checking
npm run lint      # Prettier + ESLint
```

## Components

Located in `src/lib/`:

Alert, Badge, Button, EmptyAvatar, Input, InteractivePill, Select, Slideover, Spinner, Toast, Tooltip

Plus `utils/` for shared utility functions.

## Key rules

- **This is a shared library.** Both frontends import it. Test changes against both consumers.
- **Build before checking frontends.** Frontends import from zoroark's `dist/` — run `npm run build` (or `make verify` from root) before checking frontend types.
- **No dependencies on frontends.** Zoroark must not import lugia-frontend or giratina-frontend.
- **Svelte 5 syntax.** Use runes (`$state`, `$derived`, `$effect`) and snippet-based composition, not Svelte 4 patterns.
- **Tailwind CSS.** Components use Tailwind for styling. The built CSS is exported via `@dislyze/zoroark/styles.css`.
