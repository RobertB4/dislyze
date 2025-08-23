-- name: GetUsersByTenantID :many
SELECT id, name, email, status
FROM users
WHERE tenant_id = $1
AND users.is_internal_user = false
AND users.deleted_at IS NULL
ORDER BY created_at DESC;