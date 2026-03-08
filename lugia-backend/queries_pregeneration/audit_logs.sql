-- name: InsertAuditLog :exec
INSERT INTO audit_logs (tenant_id, actor_id, resource_type, action, outcome, resource_id, metadata, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CountAuditLogs :one
SELECT COUNT(*)
FROM audit_logs al
INNER JOIN users u ON u.id = al.actor_id
WHERE al.tenant_id = @tenant_id
AND (@actor_id::uuid IS NULL OR al.actor_id = @actor_id)
AND (@resource_type::varchar = '' OR al.resource_type = @resource_type)
AND (@action::varchar = '' OR al.action = @action)
AND (@outcome::varchar = '' OR al.outcome = @outcome)
AND (@from_date::timestamptz IS NULL OR al.created_at >= @from_date)
AND (@to_date::timestamptz IS NULL OR al.created_at <= @to_date);

-- name: ListAuditLogs :many
SELECT
    al.id,
    al.tenant_id,
    al.actor_id,
    u.name AS actor_name,
    u.email AS actor_email,
    al.resource_type,
    al.action,
    al.outcome,
    al.resource_id,
    al.metadata,
    al.ip_address,
    al.user_agent,
    al.created_at
FROM audit_logs al
INNER JOIN users u ON u.id = al.actor_id
WHERE al.tenant_id = @tenant_id
AND (@actor_id::uuid IS NULL OR al.actor_id = @actor_id)
AND (@resource_type::varchar = '' OR al.resource_type = @resource_type)
AND (@action::varchar = '' OR al.action = @action)
AND (@outcome::varchar = '' OR al.outcome = @outcome)
AND (@from_date::timestamptz IS NULL OR al.created_at >= @from_date)
AND (@to_date::timestamptz IS NULL OR al.created_at <= @to_date)
ORDER BY al.created_at DESC
LIMIT @limit_count OFFSET @offset_count;
