-- name: GetUsersByTenantID :many
SELECT id, email, name, role, status, created_at, updated_at
FROM users
WHERE tenant_id = $1
ORDER BY created_at ASC; 

-- name: InviteUserToTenant :one
INSERT INTO users (tenant_id, email, password_hash, name, role, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id; 

-- name: CreateInvitationToken :one
INSERT INTO invitation_tokens (token_hash, tenant_id, user_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInvitationByTokenHash :one
SELECT * FROM invitation_tokens
WHERE token_hash = $1 AND expires_at > CURRENT_TIMESTAMP;
