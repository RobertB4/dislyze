# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in lugia-backend, a server written in golang.

## Essential Commands
```bash
make build            # Build Go binary
make test-unit        # Run unit tests
make test-integration # Run integration tests (Docker required)
make lint             # Run golangci-lint
make sqlc             # Generate SQL queries from queries_pregeneration/*.sql
```

## Architecture
- `features/`: HTTP request handlers (e.g. auth, users)
- `lib/`: Core utilities
  - `middleware/`: Auth and authorization middleware
  - `responder/`: Standardized HTTP responses
  - etc...
- `queries/`: SQLC-generated database queries
- `queries_pregeneration/`: SQL source files for SQLC

## Testing Strategy

1. **Unit Tests**: Test individual, pure functions.
2. **Integration Tests**: Test API endpoints with real database (Docker)

Run unit tests:
```bash
# Backend
make test-unit
```

Run integration test:
```bash
make test-integration
```

## Code Patterns and Conventions

### Validation Pattern
All backend requests have a `Validate()` method that:
1. Trims whitespace
2. Checks required fields
3. Validates formats
4. Returns specific error messages for logging

### Backend Error Handling
```go
// Use errlib for all errors
appErr := errlib.New(err, http.StatusBadRequest, "")
responder.RespondWithError(w, appErr)
```

### API Response Formats
- Success: direct data for queries (get requests), status 200 without body for mutations (post etc), unless returning data is explicitly required. Most of the time, we can refetch the get request instead.
- Error: `{"error": "user-friendly message"}`. 
  → User friendly error messages are only needed in cases where the info requires server knowledge. E.g. "このメールアドレスは既に使用されています。"
- Lists: Include pagination metadata

### User-Facing Messages
- Use Japanese for all user-facing error messages
- Log technical details internally
- Show friendly messages to users, but only in cases where the info requires server knowledge. E.g. "このメールアドレスは既に使用されています。"