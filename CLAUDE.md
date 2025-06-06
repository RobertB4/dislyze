# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a full-stack multi-tenant SaaS application with:
- **Backend**: Go HTTP server using Chi router, PostgreSQL, and SQLC
- **Frontend**: SvelteKit with TypeScript, Tailwind CSS, and Svelte 5
- **Email Mock**: SendGrid mock server for development

## Essential Commands

### Development
```bash
make dev              # Run all services (backend, frontend, sendgrid-mock)
make dev-backend      # Backend only with hot reload
make dev-frontend     # Frontend only
```

### Backend Commands
```bash
make build            # Build Go binary
make test-unit        # Run unit tests
make test-integration # Run integration tests (Docker required)
make lint             # Run golangci-lint
make sqlc             # Generate SQL queries from queries_pregeneration/*.sql
```

### Frontend Commands
```bash
npm run dev           # Development server
npm run build         # Production build
npm run test-e2e      # E2E tests (Playwright)
npm run lint          # ESLint and Prettier
npm run format        # Format code
```

### Database Commands
```bash
make migrate          # Run database migrations
make initdb           # Drop and recreate database with migrations
```

### Frontend Format
Follow the format specified in ./lugia-frontend/.prettierrc

## Architecture

### Backend Structure
- `handlers/`: HTTP request handlers (auth.go, users.go)
- `lib/`: Core utilities
  - `db/`: Database connection and migrations
  - `jwt/`: JWT token management
  - `middleware/`: Auth and authorization middleware
  - `responder/`: Standardized HTTP responses
- `queries/`: SQLC-generated database queries
- `queries_pregeneration/`: SQL source files for SQLC

### Frontend Structure
- `routes/`: SvelteKit file-based routing
  - `auth/`: Authentication pages (login, signup, reset-password)
  - `settings/`: User management
- `components/`: Reusable UI components
- `lib/`: Utilities (fetch wrapper, error handling, routing)

### Key Features
- JWT authentication with refresh tokens
- Multi-tenant architecture (tenants and users)
- Role-based access (admin/editor)
- User invitations and email verification
- Password reset flow
- Email change with verification
- Rate limiting on auth endpoints

## Testing Strategy

1. **Unit Tests**: Test individual functions and components
2. **Integration Tests**: Test API endpoints with real database (Docker)
3. **E2E Tests**: Test full user flows with Playwright

Run a single test:
```bash
# Backend
go test -run TestSpecificFunction ./handlers

## Environment Variables

Backend requires:
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL connection
- `JWT_SECRET`: JWT signing key
- `SENDGRID_API_KEY`: SendGrid API key
- `FRONTEND_URL`: Frontend URL for email links

Frontend requires:
- `PUBLIC_API_URL`: Backend API URL

## Database Schema

Multi-tenant structure with:
- `tenants`: Organization accounts with plan tiers
- `users`: User accounts with roles and statuses
- `refresh_tokens`: JWT refresh token storage
- `password_reset_tokens`: Password reset flow
- `invitation_tokens`: User invitation system
- `email_change_tokens`: Email change verification

## Code Patterns and Conventions

### Backend Error Handling
```go
// Use errlib for all errors
appErr := errlib.New(err, http.StatusBadRequest, "ユーザー名は必須です")
responder.RespondWithError(w, appErr)
```

### API Response Formats
- Success: direct data for queries (get requests), status 200 without body for mutations (post etc), unless returning data is explicitly required. Most of the time, we can refetch the get request instead.
- Error: `{"error": "user-friendly message"}`. 
  → User friendly error messages are only needed in cases where the info requires server knowledge. E.g. "このメールアドレスは既に使用されています。"
- Lists: Include pagination metadata

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

### Validation Pattern
All backend requests have a `Validate()` method that:
1. Trims whitespace
2. Checks required fields
3. Validates formats
4. Returns specific error messages

### User-Facing Messages
- Use Japanese for all user-facing error messages
- Log technical details internally
- Show friendly messages to users, but only in cases where the info requires server knowledge. E.g. "このメールアドレスは既に使用されています。"

## General guidelines

### Accuracy over speed
We prioritize writing correct code over writing code fast. This means we want to:
- Come up with an implementation plan before writing code
- Correctly understand the problem before writing code
- Proactively ask claryfing questions and communicate unknowns/risks before writing code

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