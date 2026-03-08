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

func RequirePermission(db *queries.Queries, resource authz.Resource, action string) func(http.Handler) http.Handler {
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
					Resource:  resource.String(),
					Action:    action,
				})

				if authz.TenantHasFeature(r.Context(), authz.FeatureAuditLog) {
					metadata, _ := json.Marshal(map[string]string{
						"resource": resource.String(),
						"action":   action,
					})
					ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
					//nolint:auditcheck // access already denied (403), audit log is best-effort for denial events
					if err := db.InsertAuditLog(r.Context(), &queries.InsertAuditLogParams{
						TenantID:     tenantID,
						ActorID:      userID,
						ResourceType: string(auditlog.ResourceAccess),
						Action:       string(auditlog.ActionPermissionDenied),
						Outcome:      string(auditlog.OutcomeFailure),
						ResourceID:   pgtype.Text{},
						Metadata:     metadata,
						IpAddress:    &ipAddr,
						UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
					}); err != nil {
						errlib.LogError(fmt.Errorf("RequirePermission: failed to insert audit log: %w", err))
					}
				}

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

func RequireIPWhitelistView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceIPWhitelist, authz.ActionView)
}

func RequireIPWhitelistEdit(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceIPWhitelist, authz.ActionEdit)
}

func RequireAuditLogView(db *queries.Queries) func(http.Handler) http.Handler {
	return RequirePermission(db, authz.ResourceAuditLog, authz.ActionView)
}
