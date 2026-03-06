# RBAC (Role-Based Access Control)

Enterprise feature allowing customers to create custom roles with granular permissions. Large companies expect fine-grained access control.

## Design intent

Allow tenants to customize who can do what. When RBAC is off, tenants still have default roles (e.g. viewer) — this keeps the permission system uniform regardless of RBAC status.

## Interactions with other features

- **Enterprise feature flag:** Must be enabled per tenant by admins in giratina.
- **Touches everything:** RBAC gates access to all other features. Permission checks (`RequireUsersEdit`, `RequireRolesView`, etc.) run as middleware on protected routes.
- **IP whitelisting, user management, profile:** UI sections are shown/hidden based on the user's effective permissions.

## Non-obvious constraints

- **Default roles exist for all tenants, regardless of RBAC status.** When RBAC is off, only default roles are used. Custom roles cannot be created or assigned.
- **Custom role assignments persist when RBAC is turned off.** The `user_roles` table still contains custom role entries, but `UserHasPermission` and `GetUserPermissionsWithFallback` filter them out at query time (`roles.is_default = true` when `@rbac_enabled = false`). This means turning RBAC back on restores previous custom role assignments — no data loss.
- **viewer fallback:** If a user has no valid roles after RBAC filtering (e.g., they only had custom roles and RBAC was turned off), the system falls back to the default viewer role's permissions. This happens in SQL, not application code.
- **`edit` implies `view`:** The permission queries treat `edit` permission as implicitly granting `view` (`permissions.action = @action OR (@action = 'view' AND permissions.action = 'edit')`).
