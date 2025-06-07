package auth

import (
	"lugia/lib/config"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	dbConn      *pgxpool.Pool
	env         *config.Env
	rateLimiter *ratelimit.RateLimiter
	queries     *queries.Queries
}

func NewAuthHandler(dbConn *pgxpool.Pool, env *config.Env, rateLimiter *ratelimit.RateLimiter, queries *queries.Queries) *AuthHandler {
	return &AuthHandler{
		dbConn:      dbConn,
		env:         env,
		rateLimiter: rateLimiter,
		queries:     queries,
	}
}
