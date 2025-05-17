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
		var finalClaims *jwt.Claims
		var initialTokenErr error

		// 1. Try to get and validate existing access token from cookie
		accessCookie, err := r.Cookie("access_token")
		if err == nil { // Access token cookie exists
			claims, validationErr := jwt.ValidateToken(accessCookie.Value, []byte(m.env.JWTSecret))
			if validationErr == nil { // Token is valid
				finalClaims = claims
			} else {
				initialTokenErr = validationErr // Store error for logging if refresh also fails
				// Log details about the invalid/expired access token attempt
				logger.LogAuthEvent(logger.AuthEvent{
					EventType: "invalid_access_token",
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     validationErr.Error(),
					TokenType: "access",
				})
				// Proceed to refresh attempt if token was invalid (e.g., expired)
			}
		} else {
			initialTokenErr = err // No access token cookie
		}

		// 2. If no valid access token claims yet (either no cookie or existing token was invalid), try to refresh
		if finalClaims == nil {
			newClaimsFromRefresh, refreshErr := m.handleRefreshToken(w, r) // This function sets cookies on success
			if refreshErr != nil {
				// Determine which error to log for handleAuthError
				loggedErr := refreshErr
				if initialTokenErr != nil && !errors.Is(initialTokenErr, http.ErrNoCookie) {
					// If there was an initial token validation error (other than just no cookie), log that as primary cause before refresh attempt failed
					loggedErr = fmt.Errorf("initial token error: %v, followed by refresh error: %w", initialTokenErr, refreshErr)
				} else if initialTokenErr != nil {
					// If it was just ErrNoCookie, the refreshErr is more relevant.
					loggedErr = fmt.Errorf("no initial token, refresh error: %w", refreshErr)
				}
				m.handleAuthError(w, r, loggedErr) // Pass the most relevant error
				return
			}
			finalClaims = newClaimsFromRefresh // Use claims from the NEWLY issued access token
		}

		// 3. If after all attempts, we still don't have claims, then it's an auth failure.
		// This should ideally be caught by error handling above that calls return.
		if finalClaims == nil {
			m.handleAuthError(w, r, errors.New("unauthorized: no valid token established after all checks"))
			return
		}

		// 4. We have valid claims (either from initial token or from refresh). Populate context.
		ctx := context.WithValue(r.Context(), TenantIDKey, finalClaims.TenantID)
		ctx = context.WithValue(ctx, UserIDKey, finalClaims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) handleRefreshToken(w http.ResponseWriter, r *http.Request) (*jwt.Claims, error) {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		return nil, errors.New("no refresh token")
	}

	// Rate limit check
	if !m.rateLimiter.Allow(r.RemoteAddr) {
		return nil, errors.New("too many refresh attempts")
	}

	// Parse and validate refresh token from cookie to get claims
	claimsFromCookie, err := jwt.ValidateToken(refreshCookie.Value, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}

	// Start a database transaction
	tx, err := m.pool.Begin(r.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(r.Context()) // Rollback by default, commit on success

	qtx := m.db.WithTx(tx) // Use queries with transaction

	// Get the stored refresh token details from database using JTI from cookie claims
	storedRefreshToken, err := qtx.GetRefreshTokenByJTI(r.Context(), claimsFromCookie.JTI)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("refresh token not found in database") // Specific error for client
		}
		return nil, fmt.Errorf("failed to get refresh token from db: %w", err) // Internal error
	}

	// --- Security Check: Verify UserID consistency ---
	if storedRefreshToken.UserID != claimsFromCookie.UserID {
		return nil, errors.New("user ID mismatch between JWT and stored token") // Security error
	}

	// Check if refresh token is expired or revoked (according to DB record)
	if storedRefreshToken.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("refresh token expired") // Specific error for client
	}
	if storedRefreshToken.RevokedAt.Valid {
		return nil, errors.New("refresh token revoked") // Specific error for client
	}

	// --- Security Check: Prevent replay of an already used (for rotation) token ---
	if storedRefreshToken.LastUsedAt.Valid {
		return nil, errors.New("refresh token already used for rotation") // Security error, potential replay
	}

	// --- Security Step: Mark the current refresh token as used ---
	if err := qtx.UpdateRefreshTokenLastUsed(r.Context(), storedRefreshToken.Jti); err != nil {
		return nil, fmt.Errorf("failed to mark refresh token as used: %w", err) // Internal error
	}

	// Get user and tenant details for generating new tokens
	user, err := qtx.GetUserByID(r.Context(), claimsFromCookie.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found for refresh token") // Should be rare if token was valid
		}
		return nil, fmt.Errorf("failed to get user: %w", err) // Internal error
	}

	tenant, err := qtx.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant not found for user") // Should be rare
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err) // Internal error
	}

	// Generate new access token
	newAccessTokenString, newExpiresIn, newAccessTokenClaims, err := jwt.GenerateAccessToken(user.ID, tenant.ID, user.Role, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err) // Internal error
	}

	// Generate new refresh token (which includes a new JTI)
	newRefreshTokenString, newJTI, err := jwt.GenerateRefreshToken(user.ID, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err) // Internal error
	}

	// Store the new refresh token in the database
	_, err = qtx.CreateRefreshToken(r.Context(), &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        newJTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token in db: %w", err) // Internal error
	}

	// If all operations were successful, commit the transaction
	if err := tx.Commit(r.Context()); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err) // Internal error
	}

	// Set new cookies (outside transaction, as this is HTTP response)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessTokenString,
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
		DeviceInfo: storedRefreshToken.DeviceInfo.String,
		Timestamp:  time.Now(),
		Success:    true,
		TokenType:  "refresh",
		TokenID:    storedRefreshToken.ID.String(),
	})

	return newAccessTokenClaims, nil // Return new claims on success
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
