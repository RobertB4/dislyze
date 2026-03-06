# Profile Management

Standard SaaS profile page — change name, password, email, organization name.

## Non-obvious constraints

- **Email change uses a token-based verification flow.** The new email isn't applied immediately — a verification link is sent first. Security over convenience.
- **Organization name** change requires `tenant.edit` permission. The UI section is hidden entirely if the user lacks this permission.
