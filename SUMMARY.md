# IP Whitelist Feature Enhancement - Complete Implementation Plan

## Phase 1: Foundation - Active Flag Control 🏗️

### Database Schema Changes
```json
// tenants.enterprise_features JSON structure
{
  "ip_whitelist": {
    "enabled": true,  // Internal: Feature enabled for tenant (set by employees)
    "active": false   // User-controlled: Whether whitelist actively enforces
  }
}
```

### Go Helper Functions
```go
// lib/enterprise/ip_whitelist.go (new file)
func GetIPWhitelistEnabled(ctx context.Context, queries *queries.Queries, tenantID pgtype.UUID) (bool, error)
func GetIPWhitelistActive(ctx context.Context, queries *queries.Queries, tenantID pgtype.UUID) (bool, error)  
func SetIPWhitelistActive(ctx context.Context, queries *queries.Queries, tenantID pgtype.UUID, active bool) error
```

### Middleware Update
```go
// lib/middleware/ip_whitelist.go
// Update logic: enforce only when enabled=true AND active=true
```

---

## Phase 2: Basic CRUD Operations ✏️

### SQL Queries (if missing)
```sql
-- Check if UpdateIPWhitelistLabel exists, add if needed
-- name: UpdateIPWhitelistLabel :exec
UPDATE tenant_ip_whitelist 
SET label = $2 
WHERE id = $1 AND tenant_id = $3;
```

### Request/Response Structures
```go
// features/ip_whitelist/types.go
type UpdateLabelRequest struct {
    Label string `json:"label" validate:"required,max=100"`
}

type DeleteIPRequest struct {
    // Empty - ID comes from URL path
}

type DeleteIPResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message,omitempty"`
}
```

### New Endpoints
- `POST /api/ip-whitelist/{id}/label` - Update IP rule label
- `POST /api/ip-whitelist/{id}/delete` - Delete single IP rule

---

## Phase 3: Safe Activation/Deactivation 🔒

### Request/Response Structures
```go
// features/ip_whitelist/types.go
type ActivateRequest struct {
    Force bool `json:"force,omitempty"`
}

type ActivateResponse struct {
    Success bool   `json:"success"`
    Warning string `json:"warning,omitempty"`
    UserIP  string `json:"userIP,omitempty"`
}

type DeactivateRequest struct {
    // Empty body
}

type DeactivateResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message,omitempty"`
}
```

### Lockout Validation Logic
```go
// features/ip_whitelist/validation.go
func ValidateActivationSafety(ctx context.Context, queries *queries.Queries, tenantID pgtype.UUID, userIP string) (bool, error)
// Returns: (isSafe bool, error)
```

### New Endpoints
- `POST /api/ip-whitelist/activate` - Activate with lockout validation
- `POST /api/ip-whitelist/deactivate` - Deactivate enforcement

---

## Phase 4: Emergency Recovery System 🚨

### Database Queries
```sql
-- May need to add if missing:
-- name: CreateIPWhitelistRevertToken :one
INSERT INTO ip_whitelist_revert_tokens (...)
RETURNING token_hash;

-- name: GetRevertTokenWithConfig :one  
SELECT * FROM ip_whitelist_revert_tokens 
WHERE token_hash = $1 AND expires_at > NOW() AND used_at IS NULL;
```

### Request/Response Structures
```go
// features/ip_whitelist/types.go
type CreateRevertTokenResponse struct {
    Success   bool   `json:"success"`
    TokenSent bool   `json:"tokenSent"`
    Email     string `json:"email,omitempty"`
}

type RevertRequest struct {
    // Token comes from URL path
}

type RevertResponse struct {
    Success   bool   `json:"success"`
    Message   string `json:"message"`
    Reverted  bool   `json:"reverted"`
}
```

### Emergency System Components
```go
// features/ip_whitelist/emergency.go
func CreateEmergencyRevertToken(ctx context.Context, queries *queries.Queries, tenantID pgtype.UUID) (string, error)
func SendEmergencyRevertEmail(ctx context.Context, email string, token string, env *config.Env) error
func RestoreIPWhitelistFromSnapshot(ctx context.Context, queries *queries.Queries, configSnapshot string, tenantID pgtype.UUID) error
```

### New Endpoint
- `POST /api/ip-whitelist/revert/{token}` - Execute emergency revert

---

## Route Updates (main.go)
```go
r.Route("/ip-whitelist", func(r chi.Router) {
    r.Use(middleware.RequireIPWhitelist())
    
    // Existing
    r.With(middleware.RequireIPWhitelistView(queries)).Get("/", ipWhitelistHandler.GetIPWhitelist)
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/create", ipWhitelistHandler.AddIPToWhitelist)
    
    // Phase 2 - CRUD
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/{id}/label", ipWhitelistHandler.UpdateIPLabel)
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/{id}/delete", ipWhitelistHandler.DeleteIP)
    
    // Phase 3 - Activation
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/activate", ipWhitelistHandler.ActivateWhitelist)
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/deactivate", ipWhitelistHandler.DeactivateWhitelist)
    
    // Phase 4 - Emergency (no auth required for revert)
})

// Emergency revert endpoint (outside auth middleware)
r.Post("/api/ip-whitelist/revert/{token}", ipWhitelistHandler.ExecuteEmergencyRevert)
```

## Dependencies & Integration Points
- **Email System**: Reuse existing SendGrid integration for emergency tokens
- **JSON Manipulation**: Consistent with existing `enterprise_features` handling
- **Error Handling**: Follow existing patterns in codebase
- **Validation**: Use existing validation framework
- **Logging**: Comprehensive audit logging for all operations

## Current Implementation Status

**✅ Already Implemented:**
- IP Whitelist Middleware with CIDR matching
- Database schema (`tenant_ip_whitelist`, `ip_whitelist_revert_tokens`)
- Basic endpoints: `GET /api/ip-whitelist/` and `POST /api/ip-whitelist/create`
- Comprehensive IP utilities and validation
- RBAC integration with view/edit permissions
- Emergency revert token database structure

**🔄 To Be Implemented:**
- Active flag control system
- Additional CRUD operations (update label, delete)
- Safe activation/deactivation with lockout prevention
- Emergency email system integration
- Complete emergency recovery workflow