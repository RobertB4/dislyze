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