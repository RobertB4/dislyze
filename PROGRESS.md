# Progress

Project state and context. Read this at the start of every session.

This file tracks stable project state (what's been done, known issues). Task coordination (what's in flight, who's doing what) lives in GitHub Issues and PR descriptions — not here — so multiple agents can work concurrently without conflicts.

## Recently completed

- **Harness Tier 0** — Measurement foundation: scoring rubric, sessions.js, Chart.js dashboard
- **Harness Tier 1** — 15 quick wins: Makefile targets, CI PR triggers, PR template, deny rules, CLAUDE.md knowledge layer, /review, /selfimprove, /session-review commands
- **CI fixes** — Go 1.24.13 (govulncheck), eslint 9→10 (npm audit), gosec #nosec annotations, svelte-check type imports
- **Branch protection** — GitHub ruleset on main: require PRs, squash merge only, no force push, no deletion
- **Harness Tier 2 (partial)** — Session management (2.1-2.3): startup protocol, PROGRESS.md, CLAUDE.md table-of-contents refactor. Structural tests (2.4-2.5): generated code boundary check, CLAUDE.md reference validation. New commands: /peer-review, /bigpicture. Updated /review with process checklist, /selfimprove to cover command improvements.

## Roadmap

See `docs/harness/implementation-plan.md` for the full harness roadmap (Tier 2, 3, and deferred items).

## Known issues (pre-existing)

- 21 Go lint issues in lugia-backend (errcheck + staticcheck) — not introduced by harness work
- Zoroark prettier checks generated `.svelte-kit/` and `dist/` files — needs `.prettierignore`
- Go 1.24 is EOL (Go 1.26 released Feb 2026) — upgrade needed but separate from harness work
- Jirachi `CreateTenant` query stale (missing `auth_method` column) — will fail at runtime
- Jirachi user queries missing `deleted_at IS NULL` filter — will find soft-deleted users
