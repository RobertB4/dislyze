-- name: GetUsersByTenantID :many
SELECT id, email, name, role, status, created_at, updated_at
FROM users
WHERE tenant_id = $1
ORDER BY created_at DESC; 