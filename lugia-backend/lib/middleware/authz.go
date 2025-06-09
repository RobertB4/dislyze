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

func RequireTenantUpdate(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceTenant, permissions.ActionUpdate)
}

func RequireUsersView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionView)
}

func RequireUsersCreate(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionCreate)
}

func RequireUsersUpdate(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionUpdate)
}

func RequireUsersDelete(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceUsers, permissions.ActionDelete)
}

func RequireRolesView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionView)
}

func RequireRolesCreate(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionCreate)
}

func RequireRolesUpdate(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionUpdate)
}

func RequireRolesDelete(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, permissions.ResourceRoles, permissions.ActionDelete)
}
