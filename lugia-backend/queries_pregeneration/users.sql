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
INSERT INTO users (tenant_id, email, password_hash, name, status)
VALUES ($1, $2, $3, $4, $5)
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

-- name: UserHasPermission :one
SELECT EXISTS(
    SELECT 1 FROM user_roles
    JOIN role_permissions ON user_roles.role_id = role_permissions.role_id
    JOIN permissions ON role_permissions.permission_id = permissions.id
    WHERE user_roles.user_id = $1 AND user_roles.tenant_id = $2 
    AND permissions.resource = $3 
    AND (
        permissions.action = $4 OR 
        ($4 = 'view' AND permissions.action = 'edit')
    )
);

-- name: GetUserRoleIDs :many
SELECT role_id FROM user_roles
WHERE user_id = $1 AND tenant_id = $2;

-- name: AddRolesToUser :copyfrom
INSERT INTO user_roles (user_id, role_id, tenant_id)
VALUES ($1, $2, $3);

-- name: RemoveRolesFromUser :exec
DELETE FROM user_roles
WHERE user_id = $1 AND tenant_id = $2 AND role_id = ANY($3::uuid[]);

-- name: GetUserPermissions :many
SELECT permissions.resource, permissions.action
FROM user_roles
JOIN role_permissions ON user_roles.role_id = role_permissions.role_id
JOIN permissions ON role_permissions.permission_id = permissions.id
WHERE user_roles.user_id = $1 AND user_roles.tenant_id = $2;

-- name: GetUserRolesWithDetails :many
SELECT roles.id, roles.name, roles.description
FROM user_roles
JOIN roles ON user_roles.role_id = roles.id
WHERE user_roles.user_id = $1 AND user_roles.tenant_id = $2;

-- name: GetUsersWithRoles :many
WITH paginated_users AS (
    SELECT users.id
    FROM users
    WHERE users.tenant_id = @tenant_id
    AND (
        @search_term = '' OR 
        users.name ILIKE '%' || @search_term || '%' OR 
        users.email ILIKE '%' || @search_term || '%'
    )
    ORDER BY users.created_at DESC
    LIMIT @limit_count OFFSET @offset_count
)
SELECT 
    users.id, users.email, users.name, users.status, users.created_at, users.updated_at,
    roles.id as role_id, roles.name as role_name, roles.description as role_description
FROM users
JOIN paginated_users pu ON users.id = pu.id
LEFT JOIN user_roles ON users.id = user_roles.user_id AND users.tenant_id = user_roles.tenant_id
LEFT JOIN roles ON user_roles.role_id = roles.id
ORDER BY users.created_at DESC, users.id, roles.name;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id, tenant_id)
VALUES ($1, $2, $3);
