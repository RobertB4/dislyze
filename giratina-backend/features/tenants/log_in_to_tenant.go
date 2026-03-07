// Feature doc: docs/features/tenant-impersonation.md
package tenants

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"giratina/lib/humautil"
	"giratina/lib/middleware"
	"giratina/queries"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var LogInToTenantOp = huma.Operation{
	OperationID: "log-in-to-tenant",
	Method:      http.MethodGet,
	Path:        "/tenants/{tenantID}/login",
	Summary:     "Log in to a tenant",
	Tags:        []string{"tenants"},
	DefaultStatus: http.StatusFound,
}

type LogInToTenantInput struct {
	TenantID string `path:"tenantID"`
}

type LogInToTenantOutput struct {
	Status   int
	Location string `header:"Location"`
}

func (h *TenantsHandler) getCookieDomain() string {
	if h.env.LugiaFrontendUrl == "" {
		return ""
	}

	parsedURL, err := url.Parse(h.env.LugiaFrontendUrl)
	if err != nil {
		return ""
	}

	host := parsedURL.Hostname()

	if host == "localhost" || host == "127.0.0.1" {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return "." + strings.Join(parts[len(parts)-2:], ".")
	}

	return ""
}

func (h *TenantsHandler) LogInToTenant(ctx context.Context, input *LogInToTenantInput) (*LogInToTenantOutput, error) {
	r := middleware.GetHTTPRequest(ctx)
	w := middleware.GetResponseWriter(ctx)

	var tenantID pgtype.UUID
	if err := tenantID.Scan(input.TenantID); err != nil {
		return nil, humautil.NewError(fmt.Errorf("LogInToTenant: invalid tenant ID format: %w", err), http.StatusBadRequest)
	}

	tokenPair, userID, err := h.logInToTenant(ctx, tenantID, r)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "tenant_login",
			Service:   "giratina",
			UserID:    userID,
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})
		return nil, humautil.NewError(fmt.Errorf("LogInToTenant: %w", err), http.StatusUnauthorized)
	}

	cookieDomain := h.getCookieDomain()

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Domain:   cookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Domain:   cookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "tenant_login",
		Service:   "giratina",
		UserID:    userID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	return &LogInToTenantOutput{
		Status:   http.StatusFound,
		Location: h.env.LugiaFrontendUrl,
	}, nil
}

func (h *TenantsHandler) logInToTenant(ctx context.Context, tenantID pgtype.UUID, r *http.Request) (*jwt.TokenPair, string, error) {
	user, err := h.queries.GetInternalUserByTenantID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("no internal user found for tenant")
		}
		return nil, "", fmt.Errorf("failed to get internal user: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("tenant login: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	existingToken, err := qtx.GetRefreshTokenByUserID(ctx, user.ID)
	if err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return nil, user.ID.String(), fmt.Errorf("failed to check existing refresh token: %w", err)
	}

	if !errlib.Is(err, pgx.ErrNoRows) {
		err = qtx.UpdateRefreshTokenUsed(ctx, existingToken.Jti)
		if err != nil {
			return nil, user.ID.String(), fmt.Errorf("failed to update refresh token last used: %w", err)
		}
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenantID, []byte(h.env.LugiaAuthJWTSecret))
	if err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to generate token pair: %w", err)
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to store refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tokenPair, user.ID.String(), nil
}
