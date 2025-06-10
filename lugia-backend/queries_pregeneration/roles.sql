-- name: ValidateRolesBelongToTenant :many
SELECT id FROM roles 
WHERE id = ANY($1::uuid[]) AND tenant_id = $2;

-- name: GetTenantRolesWithPermissions :many
SELECT 
    roles.id, roles.name, roles.description, roles.is_default,
    permissions.description as permission_description
FROM roles
LEFT JOIN role_permissions ON roles.id = role_permissions.role_id
LEFT JOIN permissions ON role_permissions.permission_id = permissions.id
WHERE roles.tenant_id = $1
ORDER BY roles.name, permissions.description;

-- name: GetAllPermissions :many
SELECT id, resource, action, description FROM permissions;

-- name: CreateRole :one
INSERT INTO roles (tenant_id, name, description, is_default)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, name, description, created_at, updated_at;

-- name: CreateRolePermissionsBulk :exec
INSERT INTO role_permissions (role_id, permission_id, tenant_id)
SELECT @role_id, UNNEST(@permission_ids::uuid[]), @tenant_id;

-- name: GetRoleByID :one
SELECT * FROM roles
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateRole :exec
UPDATE roles
SET name = $1, description = $2
WHERE id = $3 AND tenant_id = $4 AND is_default = false;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND tenant_id = $2;

-- name: CheckRoleNameExists :one
SELECT EXISTS(
    SELECT 1 FROM roles 
    WHERE tenant_id = $1 AND name = $2 AND id != $3
) as exists;
