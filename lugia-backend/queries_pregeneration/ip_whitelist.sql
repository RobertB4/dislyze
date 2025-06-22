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

-- name: GetIPWhitelistRuleByID :one
SELECT id, tenant_id, ip_address, label, created_by, created_at
FROM tenant_ip_whitelist
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

-- IP Whitelist Emergency Token Operations

-- name: CreateIPWhitelistEmergencyToken :one
INSERT INTO ip_whitelist_emergency_tokens (jti)
VALUES ($1)
RETURNING id, jti, used_at, created_at;

-- name: GetIPWhitelistEmergencyTokenByJTI :one
SELECT id, jti, used_at, created_at
FROM ip_whitelist_emergency_tokens
WHERE jti = $1;

-- name: MarkIPWhitelistEmergencyTokenAsUsed :exec
UPDATE ip_whitelist_emergency_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE jti = $1;




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