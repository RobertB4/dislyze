-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (
    name,
    auth_method,
    enterprise_features
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: CreateUser :one
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    name,
    status,
    is_internal_user,
    external_sso_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ExistsUserWithEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL
);

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

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens 
SET revoked_at = CURRENT_TIMESTAMP 
WHERE jti = $1;

-- name: UpdateRefreshTokenUsed :exec
UPDATE refresh_tokens 
SET used_at = CURRENT_TIMESTAMP 
WHERE jti = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL; 

-- name: DeletePasswordResetTokenByUserID :exec
DELETE FROM password_reset_tokens
WHERE user_id = $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetPasswordResetTokenByHash :one
SELECT * FROM password_reset_tokens
WHERE token_hash = $1;

-- name: MarkPasswordResetTokenAsUsed :exec
UPDATE password_reset_tokens
SET used_at = NOW()
WHERE id = $1;

-- name: UpdateTenantName :exec
UPDATE tenants
SET name = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateTenantEnterpriseFeatures :exec
UPDATE tenants
SET enterprise_features = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: GetTenantAndUserContext :one
SELECT
    tenants.enterprise_features,
    users.is_internal_user
FROM tenants
JOIN users ON users.tenant_id = tenants.id
WHERE tenants.id = @tenant_id AND users.id = @user_id AND users.deleted_at IS NULL;

-- name: GetSSOTenantByDomain :one
SELECT id, enterprise_features
FROM tenants
WHERE enterprise_features->'sso'->>'enabled' = 'true'
AND enterprise_features->'sso'->'allowed_domains' ? @domain;

-- name: CreateSSOAuthRequest :exec
INSERT INTO sso_auth_requests (request_id, tenant_id, email, expires_at)
VALUES ($1, $2, $3, $4);

-- name: DeleteSSORequestReturning :one
DELETE FROM sso_auth_requests
WHERE request_id = $1
RETURNING request_id, tenant_id, email, expires_at;

-- name: DeleteExpiredSSORequests :exec
DELETE FROM sso_auth_requests
WHERE expires_at < NOW();