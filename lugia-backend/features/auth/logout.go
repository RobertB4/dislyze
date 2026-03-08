// Feature doc: docs/features/authentication.md, docs/features/audit-logging.md
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	jirachiAuthz "dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var LogoutOp = huma.Operation{
	OperationID: "logout",
	Method:      http.MethodPost,
	Path:        "/auth/logout",
}

type LogoutInput struct{}

func (h *AuthHandler) Logout(ctx context.Context, input *LogoutInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)
	w := middleware.GetResponseWriter(ctx)

	// Extract user from JWT before revoking — needed for audit logging.
	// Logout is on the unauthenticated route group (no LoadTenantAndUserContext),
	// so we must load the user and tenant manually.
	var logoutUser *queries.User
	var logoutTenant *queries.Tenant
	if accessCookie, cookieErr := r.Cookie("dislyze_access_token"); cookieErr == nil {
		if claims, jwtErr := jwt.ValidateToken(accessCookie.Value, []byte(h.env.AuthJWTSecret)); jwtErr == nil {
			if user, userErr := h.queries.GetUserByID(ctx, claims.UserID); userErr == nil {
				logoutUser = user
				if tenant, tenantErr := h.queries.GetTenantByID(ctx, claims.TenantID); tenantErr == nil {
					logoutTenant = tenant
				}
			}
		}
	}

	// Try to revoke the refresh token before clearing cookies
	h.revokeRefreshToken(ctx, r)

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	if logoutUser != nil && logoutTenant != nil {
		var ef jirachiAuthz.EnterpriseFeatures
		if err := json.Unmarshal(logoutTenant.EnterpriseFeatures, &ef); err == nil && ef.AuditLog.Enabled {
			metadata, _ := json.Marshal(map[string]string{
				"actor_name":  logoutUser.Name,
				"actor_email": logoutUser.Email,
			})

			ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
			//nolint:auditcheck // failing logout to log it would leave the session active, which is worse for security
			err = h.queries.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
				TenantID:     logoutTenant.ID,
				ActorID:      logoutUser.ID,
				ResourceType: string(auditlog.ResourceAuth),
				Action:       string(auditlog.ActionLogout),
				Outcome:      string(auditlog.OutcomeSuccess),
				ResourceID:   pgtype.Text{},
				Metadata:     metadata,
				IpAddress:    &ipAddr,
				UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
			})
			if err != nil {
				errlib.LogError(fmt.Errorf("logout: failed to insert audit log: %w", err))
			}
		}
	}

	return nil, nil
}

func (h *AuthHandler) revokeRefreshToken(ctx context.Context, r *http.Request) {
	refreshCookie, err := r.Cookie("dislyze_refresh_token")
	if err != nil {
		// No refresh token cookie found, nothing to revoke
		return
	}

	claims, err := jwt.ValidateToken(refreshCookie.Value, []byte(h.env.AuthJWTSecret))
	if err != nil {
		// Token is invalid or expired, log but don't fail logout
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "logout_token_validation_failed",
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Success:   false,
			Error:     err.Error(),
		})
		return
	}

	err = h.queries.RevokeRefreshToken(ctx, claims.JTI)
	if err != nil {
		// Database error, log but don't fail logout
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "logout_token_revocation_failed",
			UserID:    claims.UserID.String(),
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Success:   false,
			Error:     err.Error(),
		})
		return
	}

	// Successfully revoked token
	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "logout_token_revoked",
		UserID:    claims.UserID.String(),
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Success:   true,
	})
}
