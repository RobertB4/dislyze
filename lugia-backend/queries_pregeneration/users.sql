-- name: GetUsersByTenantID :many
SELECT id, email, name, role, status, created_at, updated_at
FROM users
WHERE tenant_id = $1 
AND (
    $2 = '' OR 
    name ILIKE '%' || $2 || '%' OR 
    email ILIKE '%' || $2 || '%'
)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountUsersByTenantID :one
SELECT COUNT(*)
FROM users
WHERE tenant_id = $1 
AND (
    $2 = '' OR 
    name ILIKE '%' || $2 || '%' OR 
    email ILIKE '%' || $2 || '%'
); 

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
WHERE token_hash = $1 AND expires_at > CURRENT_TIMESTAMP AND used_at IS NULL;

-- name: ActivateInvitedUser :exec
UPDATE users
SET password_hash = $1, status = 'active', updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND status = 'pending_verification';

-- name: MarkInvitationTokenAsUsed :exec
UPDATE invitation_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteInvitationTokensByUserIDAndTenantID :exec
DELETE FROM invitation_tokens
WHERE user_id = $1 AND tenant_id = $2;

-- name: DeleteRefreshTokensByUserID :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: UpdateUserRole :exec
UPDATE users
SET role = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND tenant_id = $3;

-- name: UpdateUserName :exec
UPDATE users
SET name = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: CreateEmailChangeToken :exec
INSERT INTO email_change_tokens (
    user_id,
    new_email,
    token_hash,
    expires_at
) VALUES ($1, $2, $3, $4);

-- name: GetEmailChangeTokenByHash :one
SELECT * FROM email_change_tokens
WHERE token_hash = $1 AND used_at IS NULL;

-- name: DeleteEmailChangeTokensByUserID :exec
DELETE FROM email_change_tokens
WHERE user_id = $1;

-- name: MarkEmailChangeTokenAsUsed :exec
UPDATE email_change_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateUserEmail :exec
UPDATE users
SET email = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2;
