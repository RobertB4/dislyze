package ip_whitelist

import (
	"lugia/lib/config"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IPWhitelistHandler struct {
	dbConn *pgxpool.Pool
	q      *queries.Queries
	env    *config.Env
}

func NewIPWhitelistHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env) *IPWhitelistHandler {
	return &IPWhitelistHandler{
		dbConn: dbConn,
		q:      q,
		env:    env,
	}
}