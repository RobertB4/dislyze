-- name: GetUsersByTenantID :many
SELECT id, name, email, status
FROM users
WHERE tenant_id = $1
AND users.is_internal_user = false
ORDER BY created_at DESC;