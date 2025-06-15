package auth

import (
	"context"
	"net/http"

	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
)

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Try to revoke the refresh token before clearing cookies
	h.revokeRefreshToken(r.Context(), r)

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

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) revokeRefreshToken(ctx context.Context, r *http.Request) {
	refreshCookie, err := r.Cookie("dislyze_refresh_token")
	if err != nil {
		// No refresh token cookie found, nothing to revoke
		return
	}

	claims, err := jwt.ValidateToken(refreshCookie.Value, []byte(h.env.JWTSecret))
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
