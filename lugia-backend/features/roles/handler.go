package roles

import (
	"lugia/lib/config"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RolesHandler struct {
	dbConn *pgxpool.Pool
	q      *queries.Queries
	env    *config.Env
}

func NewRolesHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env) *RolesHandler {
	return &RolesHandler{
		dbConn: dbConn,
		q:      q,
		env:    env,
	}
}