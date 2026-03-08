// Feature doc: docs/features/audit-logging.md
package audit_logs

import (
	"lugia/lib/config"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogsHandler struct {
	dbConn *pgxpool.Pool
	q      *queries.Queries
	env    *config.Env
}

func NewAuditLogsHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env) *AuditLogsHandler {
	return &AuditLogsHandler{
		dbConn: dbConn,
		q:      q,
		env:    env,
	}
}
