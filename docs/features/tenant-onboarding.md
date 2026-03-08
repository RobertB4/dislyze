# Tenant Onboarding

Two paths to get a new tenant set up, split by customer segment.

## Design intent

- **Self-signup (lugia):** For SMBs and self-serve users. Quick, frictionless — creates tenant + first user in one step. No admin involvement needed.
- **Admin invitation (giratina):** For enterprise customers who expect hands-on support and want features like SSO. Admins generate a 48h invitation link with pre-configured settings (company name, SSO config, allowed domains). This is essentially a sales-driven flow.

Good UX for both paths, but the enterprise path prioritizes security configuration upfront.

## Interactions with other features

- **SSO:** SSO-enabled tenants can only be created via giratina invitation. There is no way to self-signup and activate SSO afterwards. The SSO config (IdP metadata URL, allowed domains) is set at invitation time.
- **Tenant impersonation:** An `is_internal_user` account is created automatically for every tenant during setup (both self-signup and giratina invite).
- **RBAC:** New tenants get default roles regardless of RBAC status. Enterprise tenants can have RBAC enabled by admins in giratina.
- **Audit logging:** Signup and accept-invite events are logged. These are the tenant's first audit log entries.

## Non-obvious constraints

- **SSO tenants cannot be created via self-signup.** SSO requires IdP configuration that only giratina can provide at invitation time.
- **Invitation links expire after 48 hours.**
- **SSO-enabled invitations skip the password step.** The frontend detects SSO from the invitation token and redirects to the SSO flow instead of showing password fields.
