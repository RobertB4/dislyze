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

## Phase 2: Basic CRUD Operations ✅ COMPLETED

### ✅ Implemented Endpoints
- `POST /api/ip-whitelist/{id}/label/update` - Update IP rule label
- `POST /api/ip-whitelist/{id}/delete` - Delete single IP rule

### ✅ Handler Implementation
```go
// features/ip_whitelist/update_ip_label.go
type UpdateLabelRequest struct {
    Label string `json:"label"` // Optional, max 100 chars, empty string clears label
}
func (h *IPWhitelistHandler) UpdateIPLabel(w http.ResponseWriter, r *http.Request)

// features/ip_whitelist/delete_ip.go  
func (h *IPWhitelistHandler) DeleteIP(w http.ResponseWriter, r *http.Request)
```

### ✅ Key Features Implemented
- **Existence validation**: Both endpoints check if IP rule exists before operation (404 if not found)
- **Tenant isolation**: Cannot update/delete IPs from other tenants (enforced via SQL WHERE clause)
- **Optional labels**: Update endpoint accepts empty labels (stored as NULL)
- **Proper error handling**: Distinguishes between `pgx.ErrNoRows` (404) and other DB errors (500)
- **REST semantics**: Returns 200 on success, no response body for mutations

### ✅ SQL Queries Added
```sql
-- name: GetIPWhitelistRuleByID :one
SELECT id, tenant_id, ip_address, label, created_by, created_at
FROM tenant_ip_whitelist
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateIPWhitelistLabel :exec  
UPDATE tenant_ip_whitelist
SET label = $1
WHERE id = $2 AND tenant_id = $3;

-- name: RemoveIPFromWhitelist :exec
DELETE FROM tenant_ip_whitelist
WHERE id = $1 AND tenant_id = $2;
```

---

## Phase 3: Safe Activation/Deactivation 🔒

### Request/Response Structures
```go
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
    
    // Phase 2 - CRUD ✅ IMPLEMENTED
    r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/{id}/label/update", ipWhitelistHandler.UpdateIPLabel)
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
- **Phase 1 Foundation**: IP Whitelist Middleware with CIDR matching
- **Database Schema**: `tenant_ip_whitelist`, `ip_whitelist_revert_tokens` tables
- **Phase 2 CRUD**: All basic CRUD operations completed
  - `GET /api/ip-whitelist/` - List IP rules
  - `POST /api/ip-whitelist/create` - Add new IP rule  
  - `POST /api/ip-whitelist/{id}/label/update` - Update IP rule label
  - `POST /api/ip-whitelist/{id}/delete` - Delete IP rule
- **Enterprise Features**: JSON structure for `ip_whitelist.enabled` and `ip_whitelist.active`
- **RBAC Integration**: `ip_whitelist.view` and `ip_whitelist.edit` permissions
- **IP Utilities**: Comprehensive CIDR validation, IPv4/IPv6 support, client IP extraction
- **Test Coverage**: 39 comprehensive integration tests covering all CRUD operations

**🔄 To Be Implemented:**
- **Phase 3**: Safe activation/deactivation with lockout prevention
- **Phase 4**: Emergency email system integration and recovery workflow

## Phase 2 Testing Status ✅

### ✅ Comprehensive Test Coverage (39 Tests Total)
- **Add IP Tests**: 19 test cases (security, validation, IPv4/IPv6, success scenarios)
- **Update Label Tests**: 12 test cases (auth, validation, label handling, tenant isolation)  
- **Delete IP Tests**: 8 test cases (auth, validation, existence checks, tenant isolation)
- **Get IP Tests**: 20+ test cases (middleware behavior, IP filtering, tenant isolation)

### ✅ Test Infrastructure Improvements
- **Simplified test setup**: Removed redundant role creation, uses existing seed users
- **Consistent patterns**: All tests follow established codebase conventions
- **Proper tenant isolation**: Verifies 404 responses for cross-tenant access attempts
- **Updated seed data**: Added IP whitelist permissions to `setup.TestPermissionsData`

### ✅ Key Testing Patterns Established
- Use `enterprise_1` (has IP whitelist permissions) for success cases
- Use `enterprise_2` (lacks IP whitelist permissions) for 403 tests  
- Use `smb_1` for tenant isolation testing
- Table-driven tests where appropriate for cleaner organization