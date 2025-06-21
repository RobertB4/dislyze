-- name: GetTenantIPWhitelist :many
SELECT id, tenant_id, ip_address, label, created_by, created_at
FROM tenant_ip_whitelist
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: AddIPToWhitelist :one
INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, label, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, ip_address, label, created_by, created_at;

-- name: RemoveIPFromWhitelist :exec
DELETE FROM tenant_ip_whitelist
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateIPWhitelistLabel :exec
UPDATE tenant_ip_whitelist
SET label = $1
WHERE id = $2 AND tenant_id = $3;

-- name: ClearTenantIPWhitelist :exec
DELETE FROM tenant_ip_whitelist
WHERE tenant_id = $1;

-- name: GetTenantIPWhitelistCIDRs :many
SELECT ip_address::text as ip_address
FROM tenant_ip_whitelist
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: CountTenantIPWhitelistRules :one
SELECT COUNT(*)
FROM tenant_ip_whitelist
WHERE tenant_id = $1;

-- name: CheckIPExists :one
SELECT EXISTS(
    SELECT 1 
    FROM tenant_ip_whitelist 
    WHERE tenant_id = $1 AND ip_address = $2
) AS exists;

-- IP Whitelist Revert Token Operations

-- name: CreateIPWhitelistRevertToken :one
INSERT INTO ip_whitelist_revert_tokens (tenant_id, token_hash, config_snapshot, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, token_hash, config_snapshot, created_by, expires_at, created_at, used_at;

-- name: GetIPWhitelistRevertTokenByHash :one
SELECT id, tenant_id, token_hash, config_snapshot, created_by, expires_at, created_at, used_at
FROM ip_whitelist_revert_tokens
WHERE token_hash = $1 
AND expires_at > CURRENT_TIMESTAMP 
AND used_at IS NULL;

-- name: MarkIPWhitelistRevertTokenAsUsed :exec
UPDATE ip_whitelist_revert_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE id = $1;


-- name: GetUsersByIPWhitelistEditPermission :many
WITH users_with_permission AS (
    SELECT DISTINCT user_roles.user_id
    FROM user_roles
    JOIN roles ON user_roles.role_id = roles.id
    JOIN role_permissions ON roles.id = role_permissions.role_id
    JOIN permissions ON role_permissions.permission_id = permissions.id
    WHERE user_roles.tenant_id = @tenant_id
    AND permissions.resource = 'ip_whitelist'
    AND permissions.action = 'edit'
    AND (
        @rbac_enabled = true OR  -- RBAC enabled: use all roles
        roles.is_default = true      -- RBAC disabled: only default roles
    )
)
SELECT users.id, users.email, users.name
FROM users
JOIN users_with_permission ON users.id = users_with_permission.user_id
WHERE users.tenant_id = @tenant_id
AND users.status = 'active';

-- Helper query to get current IP whitelist configuration for snapshots
-- name: GetTenantIPWhitelistSnapshot :many
SELECT 
    ip_address::text as ip_address,
    label,
    created_by,
    created_at
FROM tenant_ip_whitelist
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: RestoreIPWhitelistFromSnapshot :copyfrom
INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, label, created_by, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetIPWhitelistForMiddleware :many
WITH tenant_features AS (
    SELECT 
        tenants.id,
        (tenants.enterprise_features->'ip_whitelist'->>'enabled')::boolean as ip_whitelist_enabled,
        (tenants.enterprise_features->'ip_whitelist'->>'allow_internal_admin_bypass')::boolean as allow_internal_bypass
    FROM tenants WHERE tenants.id = $1
)
SELECT 
    tenant_features.ip_whitelist_enabled,
    tenant_features.allow_internal_bypass,
    tenant_ip_whitelist.ip_address::text as ip_address
FROM tenant_features
JOIN tenant_ip_whitelist ON tenant_features.id = tenant_ip_whitelist.tenant_id
WHERE tenant_features.ip_whitelist_enabled = true;