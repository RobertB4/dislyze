package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"lugia/lib/config"
	"lugia/lib/jwt"
	"lugia/lib/logger"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

type AuthMiddleware struct {
	env         *config.Env
	db          *queries.Queries
	rateLimiter *ratelimit.RateLimiter
}

func NewAuthMiddleware(env *config.Env, db *queries.Queries, rateLimiter *ratelimit.RateLimiter) *AuthMiddleware {
	return &AuthMiddleware{
		env:         env,
		db:          db,
		rateLimiter: rateLimiter,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the access token from the cookie
		accessCookie, err := r.Cookie("access_token")
		if err != nil {
			// No access token, try refresh flow
			if err := m.handleRefreshToken(w, r); err != nil {
				m.handleAuthError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Parse and validate the access token
		claims, err := jwt.ValidateToken(accessCookie.Value, []byte(m.env.JWTSecret))
		if err != nil {
			// Access token invalid/expired, try refresh flow
			if err := m.handleRefreshToken(w, r); err != nil {
				m.handleAuthError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Access token is valid, proceed with request
		ctx := context.WithValue(r.Context(), TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) handleRefreshToken(w http.ResponseWriter, r *http.Request) error {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		return errors.New("no refresh token")
	}

	// Rate limit check
	if !m.rateLimiter.Allow(r.RemoteAddr) {
		return errors.New("too many refresh attempts")
	}

	// Parse and validate refresh token to get claims
	claims, err := jwt.ValidateToken(refreshCookie.Value, []byte(m.env.JWTSecret))
	if err != nil {
		return errors.New("invalid refresh token")
	}

	// Get refresh token from database using claims.JTI
	refreshToken, err := m.db.GetRefreshTokenByJTI(r.Context(), claims.JTI)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("invalid refresh token")
		}
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Check if refresh token is expired or revoked
	if refreshToken.ExpiresAt.Time.Before(time.Now()) || refreshToken.RevokedAt.Valid {
		return errors.New("refresh token expired or revoked")
	}

	// Check if refresh token has been used before
	if refreshToken.LastUsedAt.Valid {
		return errors.New("refresh token already used")
	}

	// Get user and tenant
	user, err := m.db.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	tenant, err := m.db.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Generate new access token
	accessToken, expiresIn, err := jwt.GenerateAccessToken(user.ID, tenant.ID, user.Role, []byte(m.env.JWTSecret))
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, jti, err := jwt.GenerateRefreshToken(user.ID, []byte(m.env.JWTSecret))
	if err != nil {
		return fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store new refresh token
	_, err = m.db.CreateRefreshToken(r.Context(), &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        jti,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create new refresh token: %w", err)
	}

	// Set new cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(expiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	// Log the token refresh
	logger.LogTokenRefresh(logger.AuthEvent{
		EventType:  "token_refresh",
		UserID:     user.ID.String(),
		IPAddress:  r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		DeviceInfo: refreshToken.DeviceInfo.String,
		Timestamp:  time.Now(),
		Success:    true,
		TokenType:  "refresh",
		TokenID:    refreshToken.ID.String(),
	})

	return nil
}

func (m *AuthMiddleware) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	// Log the auth failure
	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "auth_failure",
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   false,
		Error:     err.Error(),
	})

	// Return appropriate error response
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
