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

## Lugia-Backend Specific Guidelines

### Enterprise Features Implementation
- **Context-based sharing**: Load enterprise features once via `LoadEnterpriseFeatures` middleware
- **Type safety**: Use `authz.EnterpriseFeature` constants, not strings
- **Performance**: Avoid repeated database calls for feature checks
- **Clean interfaces**: `TenantHasFeature(ctx, feature)` - no database dependencies in business logic

### Middleware Patterns
- **Order matters**: Auth → LoadEnterpriseFeatures → Feature checks → Permissions → Handler
- **Single responsibility**: Each middleware does one thing (auth, features, permissions)
- **Context propagation**: Store shared data in context, retrieve with typed getters

### Database Integration
- **SQLC first**: All database operations through generated queries
- **Use existing types**: Prefer `queries.Tenant` over custom structs
- **Combine queries**: Single query for related data (e.g. `GetIPWhitelistForMiddleware`)
- **Proper indexing**: Add indexes for performance-critical queries

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