-- name: InsertAuditLog :exec
INSERT INTO audit_logs (tenant_id, actor_id, resource_type, action, outcome, resource_id, metadata, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
