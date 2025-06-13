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

func RequireFeature(db *queries.Queries, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authz.TenantHasFeature(r.Context(), db, feature) {
				userID := libctx.GetUserID(r.Context())
				tenantID := libctx.GetTenantID(r.Context())

				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "feature",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     fmt.Sprintf("Feature not enabled: %s", feature),
					Feature:   feature,
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRBAC(db *queries.Queries) func(http.Handler) http.Handler {
	return RequireFeature(db, authz.FeatureRBAC)
}
