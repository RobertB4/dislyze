package customers

import (
	"giratina/lib/config"
	"giratina/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsersHandler struct {
	dbConn  *pgxpool.Pool
	env     *config.Env
	queries *queries.Queries
}

func NewCustomersHandler(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) *UsersHandler {
	return &UsersHandler{
		dbConn:  dbConn,
		env:     env,
		queries: queries,
	}
}
