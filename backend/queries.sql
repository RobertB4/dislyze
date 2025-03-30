-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (
    name,
    plan,
    status
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: CreateUser :one
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    name,
    role,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: ExistsUserWithEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = $1
);

-- name: GetUserRefreshToken :one
SELECT * FROM refresh_tokens 
WHERE user_id = $1 
AND revoked_at IS NULL 
AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    device_info,
    ip_address,
    expires_at
) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens 
SET revoked_at = CURRENT_TIMESTAMP 
WHERE id = $1;



-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens 
WHERE expires_at < CURRENT_TIMESTAMP 
   OR revoked_at IS NOT NULL;

-- name: UpdateRefreshTokenLastUsed :exec
UPDATE refresh_tokens 
SET last_used_at = CURRENT_TIMESTAMP 
WHERE id = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1; 