# Audit Logging

Enterprise feature that records all security-relevant activity within a tenant. Required for SOC 2 (CC6.1, CC7.2, CC8.1), ISO 27001 (A.12.4), and GDPR Article 30 compliance.

## Design intent

- **Compliance-first, not observability.** This is not a system log — it's a customer-facing, queryable record of who did what, when, and from where. System errors (500s, DB timeouts) belong in structured logs (jirachi/logger), not here.
- **Atomic with mutations.** For write operations, the audit log insert runs in the same database transaction as the mutation. If logging fails, the mutation rolls back. This guarantees no unlogged changes — a hard compliance requirement.
- **Tenant-scoped.** Each entry belongs to exactly one tenant. Queries filter by tenant ID. There is no cross-tenant view.

## Interactions with other features

- **Enterprise feature flag:** `audit_log.enabled` must be set in the tenant's `enterprise_features` JSON. Gated via `authz.TenantHasFeature(ctx, authz.FeatureAuditLog)`. When disabled, audit log code is skipped entirely — no performance cost.
- **RBAC:** Viewing audit logs requires the `audit_log view` permission. The permission check runs as middleware before the handler.
- **Authentication:** Auth events (login, logout, signup) are logged even on failure paths. Failed logins log the outcome as `failure` with the attempted email in metadata.
- **IP whitelisting:** All IP whitelist mutations (add, update, delete, activate, deactivate, emergency deactivate) are logged with the affected IP address in metadata.

## Non-obvious constraints

- **Mutations are transactional.** The audit log insert and the mutation share a database transaction. If either fails, both roll back. This applies to all write operations. Read-only operations (e.g., `get_users` list viewed) also fail the request if logging fails — compliance requires proof of every data access.
- **Auth failure logging has no transaction.** Failed logins have no mutation to be atomic with. The audit log insert runs standalone. If it fails, the login attempt still fails (the user gets an auth error regardless), so there's no compliance gap.
- **INNER JOIN on users table.** The audit log list query joins `users` to get actor names. This means entries from deleted (anonymized) users show as "Deleted User" but are still visible. However, if the user row were physically removed, the audit entry would disappear from query results. This is acceptable because we use soft deletes.
- **CSV export downloads current page only.** The frontend CSV export serializes the currently visible table rows, not the full filtered result set. This is a known limitation for large audit trails.
- **Giratina compliance gap.** Giratina (internal admin panel) logs to the customer's `audit_logs` table, gated by the customer's feature flag. This covers customer-facing compliance but does not provide an independent internal admin audit trail.
- **No recursive logging.** Viewing the audit log page is not itself logged. This avoids infinite recursion and is standard practice — the audit log viewer is a read-only compliance tool.
