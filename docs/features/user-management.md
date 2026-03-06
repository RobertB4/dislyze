# User Management

Standard SaaS user management — invite users, assign roles, manage access.

## Interactions with other features

- **RBAC:** When RBAC is enabled, users can be assigned custom roles during invitation or later via role editing. When RBAC is off, only default roles are available.
- **Tenant onboarding:** Inviting a user is essentially onboarding a new user to the tenant. The invited user receives a link to accept and set up their account.
- **Giratina:** Admins can view users within any tenant. Customer-facing user management (lugia) is separate — customers manage their own coworkers.

## Non-obvious constraints

- **Two separate user management interfaces.** Lugia lets customers manage users within their own tenant. Giratina lets our employees view users across all tenants. These are independent UIs with different capabilities.
- **User deletion is soft delete + anonymization.** `MarkUserDeletedAndAnonymize` replaces email with `id@deleted.invalid` and name with `Deleted User`, sets `deleted_at`, and invalidates the password hash. The row stays in the DB.
- **`is_internal_user` accounts are hidden from user lists.** All user queries filter with `is_internal_user = false`. Tenant admins never see the impersonation account in their user list.
