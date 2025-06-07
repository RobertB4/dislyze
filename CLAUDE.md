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