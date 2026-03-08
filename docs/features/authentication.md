# Authentication

Standard SaaS auth flow — email/password login, signup, password reset.

## Design intent

Lugia (customer facing) and giratina (internal admin panel) share the same user accounts and auth flow. Giratina requires an additional permission flag to access, but authentication itself is the same. This keeps things simple — one auth implementation, reused by both apps.

## Interactions with other features

- **SSO:** SSO configuration is set by admins at tenant invitation time, not by the customer. Once a tenant is SSO-enabled, their users authenticate through their own IdP (SAML/OIDC) instead of email/password. Keycloak is only used as a mock IdP in development.
- **Tenant onboarding:** Signup creates both a tenant and the first user account in one step. Subsequent users are added via invitations.
- **Audit logging:** All auth events are logged — login (success and failure), logout, signup, password reset, SSO ACS, accept invite. Failed logins record the attempted email and failure reason in metadata.

## Non-obvious constraints

- **Giratina access:** There is no separate admin signup or admin password reset. Accounts are created through lugia, then granted giratina access by setting `is_internal_admin = true` via direct database access.
- **`is_internal_admin` vs `is_internal_user`:** These flags sound similar but serve different purposes:
  - `is_internal_admin` — grants access to giratina (the admin app)
  - `is_internal_user` — per-tenant account used for impersonation. When an admin impersonates a tenant, the system looks up this tenant's internal user and creates a session as that user. See tenant-impersonation.md.
