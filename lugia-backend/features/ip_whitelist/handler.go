package ip_whitelist

import (
	"lugia/lib/config"
	"lugia/queries"

	"dislyze/jirachi/ratelimit"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IPWhitelistHandler struct {
	dbConn     *pgxpool.Pool
	q          *queries.Queries
	env        *config.Env
	rateLimiter *ratelimit.RateLimiter
}

func NewIPWhitelistHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env, rateLimiter *ratelimit.RateLimiter) *IPWhitelistHandler {
	return &IPWhitelistHandler{
		dbConn:     dbConn,
		q:          q,
		env:        env,
		rateLimiter: rateLimiter,
	}
}