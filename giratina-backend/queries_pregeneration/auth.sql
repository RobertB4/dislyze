-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetRefreshTokenByUserID :one
SELECT * FROM refresh_tokens 
WHERE user_id = $1 
AND revoked_at IS NULL 
AND expires_at > CURRENT_TIMESTAMP;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    jti,
    device_info,
    ip_address,
    expires_at
) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateRefreshTokenUsed :exec
UPDATE refresh_tokens 
SET used_at = CURRENT_TIMESTAMP 
WHERE jti = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens 
SET revoked_at = CURRENT_TIMESTAMP 
WHERE jti = $1;

-- name: GetInternalUserByTenantID :one
SELECT * FROM users
WHERE tenant_id = $1 AND is_internal_user = true AND deleted_at IS NULL;

