// Feature doc: docs/features/authentication.md
package auth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"giratina/lib/middleware"
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
