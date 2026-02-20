# Coding Conventions

Detailed conventions and examples for this codebase. The root CLAUDE.md contains the principles; this document provides the depth.

## How to write comments

The role of comments is to explain WHY code was written the way it was. Comments explaining what the code does are generally not needed, unless the logic is so complex it is hard to understand.

### Example of a good comment
```go
limit32, err := conversions.SafeInt32(limit)
if err != nil {
    // Fallback to safe default if conversion fails
    limit32 = 50
}
```
This is a good comment because it is not immediately obvious why the value should be set if an error occurs.

### Example of a bad comment
```go
// Create invitation token
_, err = qtx.CreateInvitationToken(ctx, &queries.CreateInvitationTokenParams{
    TokenHash: hashedTokenStr,
    TenantID:  rawTenantID,
    UserID:    createdUserID,
    ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
})
```
This comment is bad because it just explains what the next function call does. This is already obvious by reading the function name.

## Code quality examples

### Good: Simple, type-safe interface
```go
type EnterpriseFeature string
const FeatureRBAC EnterpriseFeature = "rbac"

func TenantHasFeature(ctx context.Context, feature EnterpriseFeature) bool {
    return libctx.GetEnterpriseFeatureEnabled(ctx, string(feature))
}
```

### Poor: Complex interface with unnecessary dependencies
```go
func TenantHasFeature(ctx context.Context, db *queries.Queries, feature string) bool {
    // Multiple DB calls, string parameters, complex error handling...
}
```

### Good: Use existing types
```go
func LoadEnterpriseFeatures(db *queries.Queries) func(http.Handler) http.Handler {
    tenant, err := db.GetTenantByID(ctx, tenantID) // Use queries.Tenant directly
}
```

### Poor: Create duplicate types
```go
type TenantData struct { // Unnecessary duplication of queries.Tenant
    ID   pgtype.UUID `json:"id"`
    Name string      `json:"name"`
    // ...
}
```
