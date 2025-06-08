package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"lugia/lib/config"
	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/jwt"
	"lugia/lib/logger"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
		accessCookie, err := r.Cookie("dislyze_access_token")
		if err == nil { // Access token cookie exists
			claims, validationErr := jwt.ValidateToken(accessCookie.Value, []byte(m.env.JWTSecret))
			if validationErr == nil { // Token is valid
				finalClaims = claims
			} else {
				initialTokenErr = validationErr
				logger.LogAuthEvent(logger.AuthEvent{
					EventType: "invalid_dislyze_access_token",
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
			newClaimsFromRefresh, refreshErr := m.handleRefreshToken(w, r)
			if refreshErr != nil {
				loggedErr := refreshErr
				if initialTokenErr != nil && !errors.Is(initialTokenErr, http.ErrNoCookie) {
					loggedErr = fmt.Errorf("initial token error: %v, followed by refresh error: %w", initialTokenErr, refreshErr)
				} else if initialTokenErr != nil {
					loggedErr = fmt.Errorf("no initial token, refresh error: %w", refreshErr)
				}
				m.handleAuthError(w, r, loggedErr)
				return
			}
			finalClaims = newClaimsFromRefresh
		}

		// 3. If after all attempts, we still don't have claims, then it's an auth failure.
		// This should ideally be caught by error handling above that calls return.
		if finalClaims == nil {
			m.handleAuthError(w, r, errors.New("unauthorized: no valid token established after all checks"))
			return
		}

		// 4. We have valid claims (either from initial token or from refresh). Populate context.
		ctx := context.WithValue(r.Context(), libctx.TenantIDKey, finalClaims.TenantID)
		ctx = context.WithValue(ctx, libctx.UserIDKey, finalClaims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) handleRefreshToken(w http.ResponseWriter, r *http.Request) (*jwt.Claims, error) {
	refreshCookie, err := r.Cookie("dislyze_refresh_token")
	if err != nil {
		return nil, errors.New("no refresh token")
	}

	if !m.rateLimiter.Allow(r.RemoteAddr) {
		return nil, errors.New("too many refresh attempts")
	}

	claimsFromCookie, err := jwt.ValidateToken(refreshCookie.Value, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}

	tx, err := m.pool.Begin(r.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(r.Context()); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
			errlib.LogError(fmt.Errorf("failed to rollback transaction in handleRefreshToken: %w", rErr))
		}
	}()

	qtx := m.db.WithTx(tx)

	storedRefreshToken, err := qtx.GetRefreshTokenByJTI(r.Context(), claimsFromCookie.JTI)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("refresh token not found in database")
		}
		return nil, fmt.Errorf("failed to get refresh token from db: %w", err)
	}

	// --- Security Check: Verify UserID consistency ---
	if storedRefreshToken.UserID != claimsFromCookie.UserID {
		return nil, errors.New("user ID mismatch between JWT and stored token")
	}

	if storedRefreshToken.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("refresh token expired")
	}
	if storedRefreshToken.RevokedAt.Valid {
		return nil, errors.New("refresh token revoked")
	}

	// --- Security Check: Prevent replay of an already used (for rotation) token ---
	if storedRefreshToken.UsedAt.Valid {
		return nil, errors.New("refresh token already used for rotation")
	}

	// --- Security Step: Mark the current refresh token as used ---
	if err := qtx.UpdateRefreshTokenUsed(r.Context(), storedRefreshToken.Jti); err != nil {
		return nil, fmt.Errorf("failed to mark refresh token as used: %w", err)
	}

	user, err := qtx.GetUserByID(r.Context(), claimsFromCookie.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found for refresh token")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	tenant, err := qtx.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant not found for user")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	newAccessTokenString, newExpiresIn, newAccessTokenClaims, err := jwt.GenerateAccessToken(user.ID, tenant.ID, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshTokenString, newJTI, err := jwt.GenerateRefreshToken(user.ID, []byte(m.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	_, err = qtx.CreateRefreshToken(r.Context(), &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        newJTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token in db: %w", err)
	}

	if err := tx.Commit(r.Context()); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    newAccessTokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(newExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    newRefreshTokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

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

	return newAccessTokenClaims, nil
}

func (m *AuthMiddleware) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "auth_failure",
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   false,
		Error:     err.Error(),
	})

	w.WriteHeader(http.StatusUnauthorized)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{}); encodeErr != nil {
		errlib.LogError(errlib.New(encodeErr, http.StatusInternalServerError, "failed to encode empty JSON response in handleAuthError"))
	}
}
