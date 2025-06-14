-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (
    name
) VALUES (
    $1
) RETURNING *;

-- name: CreateUser :one
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    name,
    status
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ExistsUserWithEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = $1
);

-- name: GetRefreshTokenByJTI :one
SELECT * FROM refresh_tokens 
WHERE jti = $1 
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

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;
