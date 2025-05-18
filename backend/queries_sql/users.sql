-- name: GetUsersByTenantID :many
SELECT id, email, name, role, status, created_at, updated_at
FROM users
WHERE tenant_id = $1
ORDER BY created_at ASC; 

-- name: InviteUserToTenant :exec
INSERT INTO users (tenant_id, email, password_hash, name, role, status)
VALUES ($1, $2, $3, $4, $5, $6); 