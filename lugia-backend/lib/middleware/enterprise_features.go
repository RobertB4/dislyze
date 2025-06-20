package middleware

import (
	"fmt"
	"net/http"
	"time"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/logger"
	"lugia/lib/authz"
)

func RequireFeature(feature authz.EnterpriseFeature) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authz.TenantHasFeature(r.Context(), feature) {
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
					Feature:   string(feature),
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRBAC() func(http.Handler) http.Handler {
	return RequireFeature(authz.FeatureRBAC)
}
