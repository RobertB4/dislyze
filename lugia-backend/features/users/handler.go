package users

import (
	"lugia/lib/config"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsersHandler struct {
	dbConn                  *pgxpool.Pool
	q                       *queries.Queries
	env                     *config.Env
	resendInviteRateLimiter *ratelimit.RateLimiter
	deleteUserRateLimiter   *ratelimit.RateLimiter
	changeEmailRateLimiter  *ratelimit.RateLimiter
}

func NewUsersHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env, resendInviteRateLimiter *ratelimit.RateLimiter, deleteUserRateLimiter *ratelimit.RateLimiter, changeEmailRateLimiter *ratelimit.RateLimiter) *UsersHandler {
	return &UsersHandler{
		dbConn:                  dbConn,
		q:                       q,
		env:                     env,
		resendInviteRateLimiter: resendInviteRateLimiter,
		deleteUserRateLimiter:   deleteUserRateLimiter,
		changeEmailRateLimiter:  changeEmailRateLimiter,
	}
}
