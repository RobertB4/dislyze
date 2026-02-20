# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a full-stack multi-tenant SaaS application with:
- **Backend**: Go HTTP server using Chi router, PostgreSQL, and SQLC
- **Frontend**: SvelteKit with TypeScript, Tailwind CSS, and Svelte 5
- **Database**: PostgreSQL
- **Email Mock**: SendGrid mock server for development

### Key Features
- JWT authentication with refresh tokens
- Multi-tenant architecture (tenants and users)
- Role-based access (admin/editor)
- User invitations and email verification
- Password reset flow
- Email change with verification
- Rate limiting on auth endpoints

## Architecture

```
                    ┌──────────────┐
                    │   database/  │  PostgreSQL schema + migrations (goose)
                    │   seed.sql   │  Seed data for development
                    └──────┬───────┘
                           │
                    ┌──────┴───────┐
                    │   jirachi/   │  Shared Go library (auth, authz, ctx, errlib, jwt, etc.)
                    │              │  Also has its own SQLC queries
                    └──┬───────┬───┘
                       │       │
          ┌────────────┴─┐   ┌─┴──────────────┐
          │lugia-backend │   │giratina-backend │  Go HTTP servers (Chi router, SQLC)
          │  (customer)  │   │  (internal admin)│
          └──────┬───────┘   └──────┬──────────┘
                 │                  │
                    ┌──────────────┐
                    │   zoroark/   │  Shared Svelte 5 component library
                    └──┬───────┬───┘
                       │       │
        ┌──────────────┴─┐   ┌─┴────────────────┐
        │lugia-frontend  │   │giratina-frontend  │  SvelteKit apps (TypeScript, Tailwind)
        │  (customer)    │   │  (internal admin)  │
        └────────────────┘   └───────────────────┘
```

### Module overview

| Directory | What it is | Language |
|---|---|---|
| `database/` | Schema migrations (goose), seed data, drop script | SQL |
| `jirachi/` | Shared Go library — auth, authz, context, error handling, JWT, rate limiting, SQLC queries | Go |
| `lugia-backend/` | Customer-facing API server | Go |
| `giratina-backend/` | Internal admin API server | Go |
| `zoroark/` | Shared Svelte 5 component library (Button, Input, Alert, Toast, etc.) | Svelte/TS |
| `lugia-frontend/` | Customer-facing SvelteKit app | Svelte/TS |
| `giratina-frontend/` | Internal admin SvelteKit app | Svelte/TS |
| `infrastructure/` | Pulumi IaC for GCP deployment | TypeScript |
| `sendgrid-mock/` | Mock SendGrid server for dev | Node.js |
| `keycloak-mock/` | Mock Keycloak server for SSO dev | Shell |

### Dependency direction

- Backends depend on `jirachi/` — never the other way around
- Frontends depend on `zoroark/` — never the other way around
- `jirachi/` and `zoroark/` must NOT depend on any backend or frontend module
- `database/` is standalone — migrations are shared by all Go modules

## Essential commands

```bash
make dev              # Start all services (6 processes)
make verify           # Lint + typecheck + unit test everything
make generate         # Regenerate all SQLC across all modules
make migrate          # Run database migrations
make initdb           # Drop + migrate + seed (destructive)
```

## General guidelines

### Accuracy over speed
We prioritize writing correct code over writing code fast. This means we want to:
- Come up with an implementation plan before writing code
- Correctly understand the problem before writing code
- Proactively ask claryfing questions and communicate unknowns/risks before writing code

### Task scope
- We prioritize focusing exclusively on the scope of the task at hand without making any unrelated changes
- If we find something that is unrelated to the task at hand but we think is a good change, we add comments explaining what we want to change and why

### How to write comments
- The role of comments to explain WHY code was written in the way it was written.
- Comments explaining what the code does are generally not needed, unless the logic is so complex it is hard to understand.

#### Example of a good comment
```
	limit32, err := conversions.SafeInt32(limit)
	if err != nil {
		// Fallback to safe default if conversion fails
		limit32 = 50
	}
```
This is a good comment because it is not immediately obvious why the value should be set if an error occurs.

#### Example of a bad comment
```
	// Create invitation token
	_, err = qtx.CreateInvitationToken(ctx, &queries.CreateInvitationTokenParams{
		TokenHash: hashedTokenStr,
		TenantID:  rawTenantID,
		UserID:    createdUserID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
```
This comment is bad because it just explains what the next function call does. This is already obvious by reading the function name.

### Locality of behavior over Seperation of concerns
Code that belongs together should be located closely together, e.g. in the same file or the same directory.
There are valid reasons to have code that belongs together live in another directory, but we should only seperate code if there is good reason to do so.

## Code Quality & Implementation Guidelines

### Follow existing patterns before introducing new ones
- **Study the codebase first**: Look at how similar problems are solved in the existing code
- **Use existing types**: Prefer extending or using existing structs over creating new ones
- **Match existing interfaces**: Follow established function signatures and error handling patterns
- **Consistent naming**: Follow the naming conventions already established in the codebase

### Prefer simplicity over complexity
- **Simple interfaces**: Functions should be easy to call and understand
- **Direct solutions**: Avoid over-engineering with complex error handling when simple approaches work
- **Type safety**: Use proper Go types and constants for compile-time safety
- **Minimal dependencies**: Don't add dependencies when existing code can be reused

### Performance through good design
- **Context sharing**: Use context to share data instead of repeated database calls
- **Single responsibility**: Each function should do one thing well
- **Avoid duplication**: Don't create new structs when existing database models can be used
- **Efficient queries**: Combine database operations when possible

### Examples of good vs. poor implementation choices

#### Good: Simple, type-safe interface
```go
type EnterpriseFeature string
const FeatureRBAC EnterpriseFeature = "rbac"

func TenantHasFeature(ctx context.Context, feature EnterpriseFeature) bool {
    return libctx.GetEnterpriseFeatureEnabled(ctx, string(feature))
}
```

#### Poor: Complex interface with unnecessary dependencies
```go
func TenantHasFeature(ctx context.Context, db *queries.Queries, feature string) bool {
    // Multiple DB calls, string parameters, complex error handling...
}
```

#### Good: Use existing types
```go
func LoadEnterpriseFeatures(db *queries.Queries) func(http.Handler) http.Handler {
    tenant, err := db.GetTenantByID(ctx, tenantID) // Use queries.Tenant directly
}
```

#### Poor: Create duplicate types
```go
type TenantData struct { // Unnecessary duplication of queries.Tenant
    ID   pgtype.UUID `json:"id"`
    Name string      `json:"name"`
    // ...
}
```

## Generated code boundaries

Files in `queries/` directories are generated by SQLC. **Never hand-edit them.**

To change database queries:
1. Edit the SQL files in `queries_pregeneration/` (in the relevant module)
2. Run `make generate` from the repo root (or `make sqlc` from the module)
3. Commit both the SQL source and the generated output

Generated files exist in: `jirachi/queries/`, `lugia-backend/queries/`, `giratina-backend/queries/`.

## Shared resource blast radius

Some directories are shared across multiple modules. Changes to these have a wider blast radius than changes to a single backend or frontend:

| Resource | Consumed by | Impact of changes |
|---|---|---|
| `database/migrations/` | All Go modules (lugia-backend, giratina-backend, jirachi) | Schema changes affect all backends |
| `jirachi/` | lugia-backend, giratina-backend | Library changes affect both backends |
| `zoroark/` | lugia-frontend, giratina-frontend | Component changes affect both frontends |
| `database/seed.sql` | Local dev setup | Seed changes can break local dev for all modules |

When changing a shared resource, verify all consumers still work by running `make verify`.

## Definition of Done

A task is complete when:

1. **Code works**: The feature/fix functions as intended
2. **Verification passes**: `make verify` passes from the repo root
3. **Scope is clean**: No unrelated changes in the diff
4. **Generated code is correct**: If queries changed, `make generate` was run (not hand-edited)
5. **Self-reviewed**: `/review` was run and issues addressed

## Escalation protocol

Stop and ask the human when:

- **Ambiguous requirements**: The task can be interpreted in multiple valid ways
- **Shared resource changes**: You need to modify `database/migrations/`, `jirachi/`, or `zoroark/` and you're unsure of the impact
- **Security decisions**: Authentication, authorization, or data access changes
- **Destructive operations**: Database migrations that drop columns/tables, deleting files, force-pushing
- **Scope creep**: You discover the task requires significantly more work than expected
- **Blocked**: You've tried two approaches and both failed

Do NOT ask when:
- The task is well-defined and the approach is clear
- You're following an existing pattern in the codebase
- The change is contained to a single module
