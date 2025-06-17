-- name: GetUserPermissions :many
SELECT permissions.resource, permissions.action
FROM user_roles
JOIN role_permissions ON user_roles.role_id = role_permissions.role_id
JOIN permissions ON role_permissions.permission_id = permissions.id
WHERE user_roles.user_id = $1 AND user_roles.tenant_id = $2;

-- name: GetUsersByTenantID :many
SELECT id, name, email, status
FROM users
WHERE tenant_id = $1
AND users.is_internal_user = false
ORDER BY created_at DESC;