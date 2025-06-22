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
  - `iputils/`: IP validation, CIDR handling, and client IP extraction
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

## Security Best Practices

### Authentication and Authorization
- **Database-backed authentication**: Always validate user permissions and internal status through database queries
- **Avoid header-based security**: Never use HTTP headers for authentication or authorization decisions (headers can be spoofed)
- **Internal user validation**: Use `libctx.GetIsInternalUser(ctx)` for secure internal user checks, not header inspection
- **Context-based security**: Load security-relevant data once in middleware, store in context for downstream use

### Anti-patterns to Avoid
- Header-based admin/internal user checks (e.g. `X-Internal-Admin-Key`)
- String-based feature flags instead of typed constants
- Direct database calls in business logic for repeated security checks

### IPv4/IPv6 Support
- All utilities handle both IPv4 and IPv6 automatically
- Single IPs are auto-converted to appropriate CIDR notation (/32 for IPv4, /128 for IPv6)
- Use PostgreSQL `INET` type for database storage

## Lugia-Backend Specific Guidelines

### Enterprise Features Implementation
- **Context-based sharing**: Load enterprise features once via `LoadTenantAndUserContext` middleware
- **Type safety**: Use `authz.EnterpriseFeature` constants, not strings
- **Performance**: Avoid repeated database calls for feature checks
- **Clean interfaces**: `TenantHasFeature(ctx, feature)` - no database dependencies in business logic

### Middleware Patterns
- **Order matters**: Auth → LoadTenantAndUserContext → IPWhitelistMiddleware → Feature checks → Permissions → Handler
- **Single responsibility**: Each middleware does one thing (auth, context loading, IP filtering, permissions)
- **Context propagation**: Store shared data in context, retrieve with typed getters
- **Performance optimization**: Load all request-scoped data once in `LoadTenantAndUserContext`

### Context Management
The `LoadTenantAndUserContext` middleware loads both enterprise features and user metadata in a single database query:

```go
// Access enterprise features
if authz.TenantHasFeature(ctx, authz.FeatureIPWhitelist) {
    // Feature is enabled
}

// Access user metadata
if libctx.GetIsInternalUser(ctx) {
    // User is internal, allow bypass
}

// Get specific feature configuration
ipConfig := libctx.GetIPWhitelistConfig(ctx)
```

### Database Integration
- **SQLC first**: All database operations through generated queries
- **Use existing types**: Prefer `queries.Tenant` over custom structs
- **Query optimization**: Combine related data in single queries for performance
- **Proper indexing**: Add indexes for performance-critical queries

### Query Optimization Patterns

#### Combine Related Data Fetches
```go
// Good: Single query for related data
func GetTenantAndUserContext(ctx context.Context, tenantID, userID pgtype.UUID) (*GetTenantAndUserContextRow, error)

// Bad: Multiple separate queries
func GetTenant(ctx context.Context, tenantID pgtype.UUID) (*Tenant, error)
func GetUser(ctx context.Context, userID pgtype.UUID) (*User, error)
```

#### Performance-Critical Middleware Queries
```go
// Example: GetIPWhitelistForMiddleware combines feature check + IP rules
// Returns empty result set when feature disabled or no rules configured
// Uses INNER JOIN to avoid NULL scanning issues
```

#### Avoid N+1 Query Patterns
- Use JOINs or batch queries instead of loops with individual queries
- Combine permission checks with data fetching when possible
- Load context data once per request, not per operation

### Example: Adding a New Enterprise Feature

#### 1. Extend the EnterpriseFeatures struct
```go
// jirachi/authz/enterprise_features.go
type EnterpriseFeatures struct {
    RBAC        RBAC        `json:"rbac"`
    IPWhitelist IPWhitelist `json:"ip_whitelist"`
    NewFeature  NewFeature  `json:"new_feature"`  // Add here
}

const (
    FeatureRBAC        EnterpriseFeature = "rbac"
    FeatureIPWhitelist EnterpriseFeature = "ip_whitelist"
    FeatureNewFeature  EnterpriseFeature = "new_feature"  // Add constant
)
```

#### 2. Update context helper
```go
// jirachi/ctx/ctx.go
func GetEnterpriseFeatureEnabled(ctx context.Context, featureName string) bool {
    features := GetEnterpriseFeatures(ctx)
    switch featureName {
    case "rbac":
        return features.RBAC.Enabled
    case "ip_whitelist":
        return features.IPWhitelist.Enabled
    case "new_feature":
        return features.NewFeature.Enabled  // Add case
    default:
        return false
    }
}
```

#### 3. Update feature checking
```go
// lib/authz/enterprise_features.go
func TenantHasFeature(ctx context.Context, feature EnterpriseFeature) bool {
    switch feature {
    case FeatureRBAC:
        return libctx.GetEnterpriseFeatureEnabled(ctx, "rbac")
    case FeatureIPWhitelist:
        return libctx.GetEnterpriseFeatureEnabled(ctx, "ip_whitelist")
    case FeatureNewFeature:
        return libctx.GetEnterpriseFeatureEnabled(ctx, "new_feature")  // Add case
    default:
        return false
    }
}
```

#### 4. Create middleware helper
```go
// lib/middleware/enterprise_features.go
func RequireNewFeature() func(http.Handler) http.Handler {
    return RequireFeature(authz.FeatureNewFeature)
}
```

This pattern ensures type safety, performance, and consistency across the codebase.