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
    SELECT DISTINCT ur.user_id
    FROM user_roles ur
    JOIN roles r ON ur.role_id = r.id
    JOIN role_permissions rp ON r.id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
    WHERE ur.tenant_id = @tenant_id
    AND p.resource = 'ip_whitelist'
    AND p.action = 'edit'
    AND (
        @rbac_enabled = true OR  -- RBAC enabled: use all roles
        r.is_default = true      -- RBAC disabled: only default roles
    )
)
SELECT u.id, u.email, u.name
FROM users u
JOIN users_with_permission uwp ON u.id = uwp.user_id
WHERE u.tenant_id = @tenant_id
AND u.status = 'active';

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
        t.id,
        (t.enterprise_features->>'ip_whitelist'->>'enabled')::boolean as ip_whitelist_enabled,
        (t.enterprise_features->>'ip_whitelist'->>'allow_internal_admin_bypass')::boolean as allow_internal_bypass
    FROM tenants t WHERE t.id = $1
)
SELECT 
    tf.ip_whitelist_enabled,
    tf.allow_internal_bypass,
    tiw.ip_address::text as ip_address
FROM tenant_features tf
LEFT JOIN tenant_ip_whitelist tiw ON tf.id = tiw.tenant_id AND tf.ip_whitelist_enabled = true;