# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in giratina-backend, an internal admin application server written in Go.

## Project Overview

This is an internal admin application for managing customers, subscriptions, and other business operations. Built with:
- **Backend**: Go HTTP server using Chi router, PostgreSQL, and SQLC
- **Database**: PostgreSQL
- **Architecture**: Similar to lugia-backend but focused on admin operations

## Essential Commands
```bash
make test-unit    # Run unit tests
make test-integration # Run integration tests
make sqlc         # Generate SQL queries from queries_pregeneration/*.sql
```

## Architecture
- `features/`: HTTP request handlers organized by domain (e.g. customers, subscriptions)
- `lib/`: Core utilities
  - `config/`: Environment configuration
  - `db/`: Database connection and migrations
  - `middleware/`: HTTP middleware (to be added as needed)
  - `responder/`: Standardized HTTP responses (to be added as needed)
- `queries/`: SQLC-generated database queries
- `queries_pregeneration/`: SQL source files for SQLC

## Development Setup

1. Set up environment variables in `.env` file
2. Ensure PostgreSQL is running
3. Run `make dev` to start the server with hot reload

## Code Patterns and Conventions

### Handler Pattern
All handlers follow the same pattern:
```go
type FeatureHandler struct {
    dbConn  *pgxpool.Pool
    env     *config.Env
    queries *queries.Queries
}

func NewFeatureHandler(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) *FeatureHandler {
    return &FeatureHandler{...}
}
```

### Database Queries
- All database queries are defined in `queries_pregeneration/*.sql`
- Run `make sqlc` to generate Go code from SQL
- Use the generated queries in handlers

### Environment Configuration
- All environment variables are defined in `lib/config/env.go`
- Load environment variables using `config.LoadEnv()`

## Next Steps
This is a basic scaffold. The following can be added as needed:
- Authentication middleware
- Error handling utilities
- Response formatting utilities
- Logging
- Rate limiting
- Additional features (billing, analytics, etc.)