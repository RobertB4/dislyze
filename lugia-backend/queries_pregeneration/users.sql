-- name: CountUsersByTenantID :one
SELECT COUNT(*)
FROM users
WHERE tenant_id = $1 
AND is_internal_user = false
AND deleted_at IS NULL
AND (
    $2 = '' OR 
    name ILIKE '%' || $2 || '%' OR 
    email ILIKE '%' || $2 || '%'
); 

-- name: InviteUserToTenant :one
INSERT INTO users (tenant_id, email, password_hash, name, status, auth_method, external_sso_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
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

-- name: MarkUserDeletedAndAnonymize :exec
UPDATE users 
SET 
    email = id::text || '@deleted.invalid',
    name = '削除済みユーザー',
    password_hash = '$2a$10$invalidhashthatshouldnevermatchanythingever',
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
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
WITH user_permissions AS (
  -- Get permissions from user's assigned roles (filtered by RBAC status)
  SELECT 1 as found
  FROM user_roles
  JOIN roles ON user_roles.role_id = roles.id
  JOIN role_permissions ON user_roles.role_id = role_permissions.role_id
  JOIN permissions ON role_permissions.permission_id = permissions.id
  WHERE user_roles.user_id = @user_id 
    AND user_roles.tenant_id = @tenant_id
    AND permissions.resource = @resource
    AND (
        permissions.action = @action OR 
        (@action = 'view' AND permissions.action = 'edit')
    )
    AND (
      @rbac_enabled = true OR  -- RBAC enabled: use all roles
      roles.is_default = true  -- RBAC disabled: only default roles
    )
  LIMIT 1
),
fallback_permissions AS (
  -- Fallback: Check 閲覧者 permissions if user has no valid roles
  SELECT 1 as found
  FROM roles
  JOIN role_permissions ON roles.id = role_permissions.role_id
  JOIN permissions ON role_permissions.permission_id = permissions.id
  WHERE roles.tenant_id = @tenant_id
    AND roles.name = '閲覧者'
    AND roles.is_default = true
    AND permissions.resource = @resource
    AND (
        permissions.action = @action OR 
        (@action = 'view' AND permissions.action = 'edit')
    )
    AND NOT EXISTS (SELECT 1 FROM user_permissions)
  LIMIT 1
)
SELECT EXISTS(
    SELECT 1 FROM user_permissions
    UNION ALL
    SELECT 1 FROM fallback_permissions
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

-- name: GetUserPermissionsWithFallback :many
WITH user_permissions AS (
  -- Get permissions from user's assigned roles (filtered by RBAC status)
  SELECT DISTINCT permissions.resource, permissions.action, 1 as priority
  FROM user_roles
  JOIN roles ON user_roles.role_id = roles.id
  JOIN role_permissions ON user_roles.role_id = role_permissions.role_id
  JOIN permissions ON role_permissions.permission_id = permissions.id
  WHERE user_roles.user_id = @user_id 
    AND user_roles.tenant_id = @tenant_id
    AND (
      @rbac_enabled = true OR  -- RBAC enabled: use all roles
      roles.is_default = true  -- RBAC disabled: only default roles
    )
),
fallback_permissions AS (
  -- Fallback: Get 閲覧者 permissions if user has no valid roles
  SELECT DISTINCT permissions.resource, permissions.action, 2 as priority
  FROM roles
  JOIN role_permissions ON roles.id = role_permissions.role_id
  JOIN permissions ON role_permissions.permission_id = permissions.id
  WHERE roles.tenant_id = @tenant_id
    AND roles.name = '閲覧者'
    AND roles.is_default = true
    AND NOT EXISTS (SELECT 1 FROM user_permissions)
)
SELECT resource, action
FROM (
  SELECT resource, action, priority FROM user_permissions
  UNION ALL
  SELECT resource, action, priority FROM fallback_permissions
) combined
ORDER BY priority, resource, action;

-- name: GetUserRolesWithDetails :many
SELECT roles.id, roles.name, roles.description
FROM user_roles
JOIN roles ON user_roles.role_id = roles.id
WHERE user_roles.user_id = $1 AND user_roles.tenant_id = $2;

-- name: GetUsersWithRolesRespectingRBAC :many
WITH paginated_users AS (
    SELECT users.id
    FROM users
    WHERE users.tenant_id = @tenant_id
    AND users.is_internal_user = false
    AND users.deleted_at IS NULL
    AND (
        @search_term = '' OR 
        users.name ILIKE '%' || @search_term || '%' OR 
        users.email ILIKE '%' || @search_term || '%'
    )
    ORDER BY users.created_at DESC
    LIMIT @limit_count OFFSET @offset_count
),
user_roles_with_rbac AS (
    SELECT DISTINCT 
        users.id as user_id,
        roles.id as role_id, 
        roles.name as role_name, 
        roles.description as role_description,
        1 as priority
    FROM users
    JOIN paginated_users pu ON users.id = pu.id
    JOIN user_roles ON users.id = user_roles.user_id AND users.tenant_id = user_roles.tenant_id
    JOIN roles ON user_roles.role_id = roles.id
    WHERE (
        @rbac_enabled = true OR  -- RBAC enabled: use all roles
        roles.is_default = true  -- RBAC disabled: only default roles
    )
),
fallback_roles AS (
    SELECT DISTINCT 
        users.id as user_id,
        roles.id as role_id, 
        roles.name as role_name, 
        roles.description as role_description,
        2 as priority
    FROM users
    JOIN paginated_users pu ON users.id = pu.id
    JOIN roles ON roles.tenant_id = @tenant_id
    WHERE roles.name = '閲覧者'
    AND roles.is_default = true
    AND NOT EXISTS (
        SELECT 1 FROM user_roles_with_rbac urwr 
        WHERE urwr.user_id = users.id
    )
)
SELECT 
    users.id, users.email, users.name, users.status, users.created_at, users.updated_at,
    combined_roles.role_id, combined_roles.role_name, combined_roles.role_description
FROM users
JOIN paginated_users pu ON users.id = pu.id
LEFT JOIN (
    SELECT user_id, role_id, role_name, role_description, priority 
    FROM user_roles_with_rbac
    UNION ALL
    SELECT user_id, role_id, role_name, role_description, priority 
    FROM fallback_roles
) combined_roles ON users.id = combined_roles.user_id
ORDER BY users.created_at DESC, users.id, combined_roles.priority, combined_roles.role_name;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id, tenant_id)
VALUES ($1, $2, $3);
