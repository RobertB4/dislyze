package middleware

import (
	"fmt"
	"net/http"

	"lugia/lib/errlib"
	"lugia/lib/permissions"
	"lugia/queries"
)

func RequirePermission(db *queries.Queries, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !permissions.UserHasPermission(r.Context(), db, resource, action) {
				errlib.LogError(fmt.Errorf("Forbidden: Permission required. resource: %s, action: %s", resource, action))
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireTenantEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceTenant, permissions.ActionEdit)
}

func RequireUsersView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionView)
}

func RequireUsersEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionEdit)
}

func RequireRolesView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionView)
}

func RequireRolesEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionEdit)
}
