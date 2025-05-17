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
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool        *pgxpool.Pool
}

func NewAuthMiddleware(env *config.Env, db *queries.Queries, rateLimiter *ratelimit.RateLimiter, pool *pgxpool.Pool) *AuthMiddleware {
	return &AuthMiddleware{
		env:         env,
		db:          db,
		rateLimiter: rateLimiter,
		pool:        pool,
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

	// Parse and validate refresh token from cookie to get claims
	claims, err := jwt.ValidateToken(refreshCookie.Value, []byte(m.env.JWTSecret))
	if err != nil {
		// This could be due to an invalid signature, or if the token is malformed/expired by JWT standards.
		return errors.New("invalid refresh token signature or format")
	}

	// Start a database transaction
	tx, err := m.pool.Begin(r.Context())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(r.Context()) // Rollback by default, commit on success

	qtx := m.db.WithTx(tx) // Use queries with transaction

	// Get the stored refresh token details from database using JTI from cookie claims
	storedRefreshToken, err := qtx.GetRefreshTokenByJTI(r.Context(), claims.JTI)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("refresh token not found in database")
		}
		return fmt.Errorf("failed to get refresh token from db: %w", err)
	}

	// --- Security Check: Verify UserID consistency ---
	if storedRefreshToken.UserID != claims.UserID {
		// This is a critical security check. If the UserID in the JWT doesn't match the one associated with the JTI in the DB,
		// it could indicate a compromised JWT or a serious flaw.
		return errors.New("user ID mismatch between JWT and stored token")
	}

	// Check if refresh token is expired or revoked (according to DB record)
	if storedRefreshToken.ExpiresAt.Time.Before(time.Now()) {
		return errors.New("refresh token expired")
	}
	if storedRefreshToken.RevokedAt.Valid {
		return errors.New("refresh token revoked")
	}

	// --- Security Check: Prevent replay of an already used (for rotation) token ---
	if storedRefreshToken.LastUsedAt.Valid {
		// If LastUsedAt is set, this token has already been used to rotate tokens.
		// This is a critical defense against replay attacks of a token that was part of a successful rotation.
		// Consider revoking all tokens for this user if this happens, as it indicates a potential compromise.
		return errors.New("refresh token already used for rotation")
	}

	// --- Security Step: Mark the current refresh token as used ---
	if err := qtx.UpdateRefreshTokenLastUsed(r.Context(), storedRefreshToken.Jti); err != nil {
		return fmt.Errorf("failed to mark refresh token as used: %w", err)
	}

	// Get user and tenant details for generating new tokens
	user, err := qtx.GetUserByID(r.Context(), claims.UserID) // Use claims.UserID as it's now verified against storedRefreshToken.UserID
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user not found for refresh token")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	tenant, err := qtx.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant not found for user")
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Generate new access token
	newAccessToken, newExpiresIn, err := jwt.GenerateAccessToken(user.ID, tenant.ID, user.Role, []byte(m.env.JWTSecret))
	if err != nil {
		return fmt.Errorf("failed to generate new access token: %w", err)
	}

	// Generate new refresh token (which includes a new JTI)
	newRefreshTokenString, newJTI, err := jwt.GenerateRefreshToken(user.ID, []byte(m.env.JWTSecret))
	if err != nil {
		return fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Store the new refresh token in the database
	_, err = qtx.CreateRefreshToken(r.Context(), &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        newJTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true}, // Consider copying from old token or updating
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},  // Consider copying from old token or updating
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create new refresh token in db: %w", err)
	}

	// If all operations were successful, commit the transaction
	if err := tx.Commit(r.Context()); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Set new cookies (outside transaction, as this is HTTP response)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(newExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshTokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	// Log the token refresh (outside transaction)
	logger.LogTokenRefresh(logger.AuthEvent{
		EventType:  "token_refresh_successful",
		UserID:     user.ID.String(),
		IPAddress:  r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		DeviceInfo: storedRefreshToken.DeviceInfo.String, // Info from the old token
		Timestamp:  time.Now(),
		Success:    true,
		TokenType:  "refresh",
		TokenID:    storedRefreshToken.ID.String(), // ID of the old, now used token
	})

	return nil // Success
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
