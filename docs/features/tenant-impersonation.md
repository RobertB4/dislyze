# Tenant Impersonation

Allows internal admins to log in to a customer's environment to debug issues in their tenant.

## Design intent

Reuses the regular JWT auth system rather than a separate admin backdoor. The system looks up the tenant's `is_internal_user` account, generates standard JWT tokens for it, and sets cookies — the admin is then a regular user session in that tenant. This keeps the impersonation path simple and auditable.

## Interactions with other features

- **Authentication:** Uses the same JWT token flow as regular login. The impersonation session is indistinguishable from a normal session at the middleware level.
- **IP whitelisting:** The `is_internal_user` can bypass IP whitelist restrictions, but only if `AllowInternalAdminBypass` is enabled in the tenant's enterprise feature config. The customer decides whether to allow this, but the setting is managed through giratina by our admins.
- **Audit logging:** Both successful and failed impersonation attempts are logged to the structured logger (jirachi/logger), but not to the tenant's `audit_logs` table. This is a known limitation — impersonation logging exists for internal ops visibility but is not part of the customer-facing compliance audit trail.

## Non-obvious constraints

- **Every tenant has exactly one `is_internal_user` account.** This account is created during tenant setup and is used exclusively for impersonation. It is not a real user account.
- **Giratina requires the admin to type the tenant name before impersonating.** This is a UX safeguard to prevent accidental impersonation, not a security mechanism.
- **No separate admin signup or password reset for giratina.** See authentication.md — admin accounts are regular lugia accounts with `is_internal_admin = true` set via direct DB access.
