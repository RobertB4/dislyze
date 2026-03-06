// Feature doc: docs/features/tenant-onboarding.md, docs/features/tenant-impersonation.md
package tenants

import (
	"giratina/lib/config"
	"giratina/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantsHandler struct {
	dbConn  *pgxpool.Pool
	env     *config.Env
	queries *queries.Queries
}

func NewTenantsHandler(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) *TenantsHandler {
	return &TenantsHandler{
		dbConn:  dbConn,
		env:     env,
		queries: queries,
	}
}