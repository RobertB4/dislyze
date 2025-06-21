package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/logger"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/queries"
)

func IPWhitelistMiddleware(db *queries.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantID := libctx.GetTenantID(ctx)
			userID := libctx.GetUserID(ctx)

			if !authz.TenantHasFeature(ctx, authz.FeatureIPWhitelist) {
				// Feature not enabled, continue normally
				next.ServeHTTP(w, r)
				return
			}

			active := authz.GetIPWhitelistActive(ctx)
			if !active {
				// IP whitelist not active, continue normally
				next.ServeHTTP(w, r)
				return
			}

			clientIP := iputils.ExtractClientIP(r)

			ipConfig, err := db.GetIPWhitelistForMiddleware(ctx, tenantID)
			if err != nil {
				if errlib.Is(err, pgx.ErrNoRows) {
					logger.LogAccessEvent(logger.AccessEvent{
						EventType: "ip_whitelist",
						UserID:    userID.String(),
						TenantID:  tenantID.String(),
						IPAddress: clientIP,
						UserAgent: r.UserAgent(),
						Timestamp: time.Now(),
						Success:   false,
						Error:     "IP whitelist enabled but no configuration found",
						Feature:   "ip_whitelist",
					})

					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "ip_whitelist",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: clientIP,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     "Failed to load IP whitelist configuration: " + err.Error(),
					Feature:   "ip_whitelist",
				})

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Extract configuration from first row (all rows have same config)
			if len(ipConfig) == 0 {
				// No rules configured - deny all access
				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "ip_whitelist",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: clientIP,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     "Access denied: No IP addresses configured in whitelist",
					Feature:   "ip_whitelist",
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			config := ipConfig[0]

			if config.AllowInternalBypass && libctx.GetIsInternalUser(ctx) {
				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "ip_whitelist",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: clientIP,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   true,
					Error:     "Internal user bypass enabled",
					Feature:   "ip_whitelist",
				})

				next.ServeHTTP(w, r)
				return
			}

			var allowedIPs []string
			for _, row := range ipConfig {
				if row.IpAddress != "" {
					allowedIPs = append(allowedIPs, row.IpAddress)
				}
			}

			allowed, err := iputils.IsIPInCIDRList(clientIP, allowedIPs)
			if err != nil {
				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "ip_whitelist",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: clientIP,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     "IP validation error: " + err.Error(),
					Feature:   "ip_whitelist",
				})

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !allowed {
				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "ip_whitelist",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: clientIP,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     fmt.Sprintf("Access denied: IP %s not in whitelist", clientIP),
					Feature:   "ip_whitelist",
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			logger.LogAccessEvent(logger.AccessEvent{
				EventType: "ip_whitelist",
				UserID:    userID.String(),
				TenantID:  tenantID.String(),
				IPAddress: clientIP,
				UserAgent: r.UserAgent(),
				Timestamp: time.Now(),
				Success:   true,
				Error:     "",
				Feature:   "ip_whitelist",
			})

			next.ServeHTTP(w, r)
		})
	}
}
