-- name: GetTenants :many
SELECT * FROM tenants ORDER BY created_at ASC;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1;

-- name: UpdateTenant :exec
UPDATE tenants
SET name = $1, enterprise_features = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $3;
