package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	libctx "dislyze/lib/ctx"
	"dislyze/lib/errors"
)

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userRole := libctx.GetUserRole(r.Context())

		if userRole != "admin" {
			errors.LogError(fmt.Errorf("Forbidden: Administrator access required. user_role: %s", userRole))
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}

		next.ServeHTTP(w, r)
	})
}
