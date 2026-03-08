# IP Whitelisting

Enterprise feature that restricts product access to specific IP addresses/CIDR ranges. Large companies expect this — it's table stakes for enterprise SaaS security.

## Design intent

- **Global toggle + rules list** rather than per-user restrictions. Keeps it simple for tenant admins — one switch, one list.
- **Lockout prevention:** Before activation, the frontend checks if the user's current IP is in the whitelist and warns them if not. This is a UX safeguard, not a backend enforcement.
- **Emergency deactivate:** If a user gets locked out, they can deactivate the whitelist via a token sent to their email. Email is outside our product, so it's always reachable even when the product is locked.

## Interactions with other features

- **Enterprise feature flag:** Must be enabled per tenant by admins in giratina before customers can use it.
- **Auth endpoints are exempt:** The IP check runs as middleware on every request, but auth endpoints (login, SSO, password reset) are not checked. The check happens after authentication, so users can still log in — they just can't access anything else.
- **SSO:** IP check is after the IdP redirect, not before. Users complete SSO auth first, then get blocked if their IP isn't whitelisted.
- **Audit logging:** All IP whitelist mutations are logged — activate, deactivate, emergency deactivate, add/update/delete IP rules. Metadata includes the affected IP address. Mutations and audit log inserts are atomic (same transaction).

## Non-obvious constraints

- **Immediate lockout on activation:** If a tenant enables the whitelist without including their current IP, they are locked out on the very next request. Existing sessions are not preserved — middleware blocks every non-auth request regardless of session state.
- **Emergency deactivate requires tenant admin email access.** If the admin who enabled it has lost email access too, there is no self-service recovery path.
