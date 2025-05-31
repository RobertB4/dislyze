package middleware

import (
	"fmt"
	"net/http"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/queries_pregeneration"
)

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userRole := libctx.GetUserRole(r.Context())

		if userRole != queries_pregeneration.AdminRole {
			errlib.LogError(fmt.Errorf("Forbidden: Administrator access required. user_role: %s", userRole))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
