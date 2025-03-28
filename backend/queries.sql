-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND tenant_id = $2;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (
    name,
    status,
    settings
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