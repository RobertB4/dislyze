package middleware

import (
	"fmt"
	"net/http"
	"time"

	"lugia/lib/authz"
	libctx "lugia/lib/ctx"
	"lugia/lib/logger"
	"lugia/queries"
)

func RequirePermission(db *queries.Queries, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authz.UserHasPermission(r.Context(), db, resource, action) {
				userID := libctx.GetUserID(r.Context())
				tenantID := libctx.GetTenantID(r.Context())

				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "permission",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     fmt.Sprintf("Permission required. resource: %s, action: %s", resource, action),
					Resource:  resource,
					Action:    action,
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireTenantEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceTenant, authz.ActionEdit)
}

func RequireUsersView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceUsers, authz.ActionView)
}

func RequireUsersEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceUsers, authz.ActionEdit)
}

func RequireRolesView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceRoles, authz.ActionView)
}

func RequireRolesEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceRoles, authz.ActionEdit)
}
