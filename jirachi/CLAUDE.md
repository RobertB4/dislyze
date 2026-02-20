# jirachi — Shared Go Library

Jirachi is the shared Go library used by both `lugia-backend` and `giratina-backend`. Changes here affect both backends.

## Essential commands

```bash
make sqlc       # Regenerate SQLC queries from queries_pregeneration/
make test-unit  # Run unit tests
```

## Package overview

| Package | Purpose |
|---|---|
| `auth/` | Auth middleware and config |
| `authz/` | Enterprise feature definitions and authorization helpers |
| `ctx/` | Context getters/setters for request-scoped data |
| `errlib/` | Standardized application error type |
| `jwt/` | JWT token creation and validation |
| `logger/` | Structured logging setup |
| `queries/` | **Generated** — SQLC output, do not hand-edit |
| `queries_pregeneration/` | SQL source files for SQLC |
| `ratelimit/` | Rate limiting middleware |
| `responder/` | Standardized HTTP response helpers |
| `sendgridlib/` | SendGrid email client wrapper |
| `utils/` | UUID conversion utilities |

## Key rules

- **This is a shared library.** Both backends import it. Test changes against both consumers.
- **`queries/` is generated.** Edit `queries_pregeneration/*.sql`, then run `make sqlc`.
- **No dependencies on backends.** Jirachi must not import lugia-backend or giratina-backend.
- **Keep packages focused.** Each package provides one capability. Don't add unrelated functions to existing packages.
