package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/logger"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/queries"
)

func RequireFeature(feature authz.EnterpriseFeature, db *queries.Queries) func(http.Handler) http.Handler {
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

				if authz.TenantHasFeature(r.Context(), authz.FeatureAuditLog) {
					metadata, _ := json.Marshal(map[string]string{
						"feature": string(feature),
					})
					ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
					//nolint:auditcheck // access already denied (403), audit log is best-effort for denial events
					if err := db.InsertAuditLog(r.Context(), &queries.InsertAuditLogParams{
						TenantID:     tenantID,
						ActorID:      userID,
						ResourceType: string(auditlog.ResourceAccess),
						Action:       string(auditlog.ActionFeatureGateBlocked),
						Outcome:      string(auditlog.OutcomeFailure),
						ResourceID:   pgtype.Text{},
						Metadata:     metadata,
						IpAddress:    &ipAddr,
						UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
					}); err != nil {
						errlib.LogError(fmt.Errorf("RequireFeature: failed to insert audit log: %w", err))
					}
				}

				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRBAC(db *queries.Queries) func(http.Handler) http.Handler {
	return RequireFeature(authz.FeatureRBAC, db)
}

func RequireIPWhitelist(db *queries.Queries) func(http.Handler) http.Handler {
	return RequireFeature(authz.FeatureIPWhitelist, db)
}

func RequireAuditLog(db *queries.Queries) func(http.Handler) http.Handler {
	return RequireFeature(authz.FeatureAuditLog, db)
}
