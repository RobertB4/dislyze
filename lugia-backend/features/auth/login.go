// Feature doc: docs/features/authentication.md, docs/features/audit-logging.md
package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/auditlog"
	jirachiAuthz "dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var LoginOp = huma.Operation{
	OperationID: "login",
	Method:      http.MethodPost,
	Path:        "/auth/login",
}

type LoginInput struct {
	Body LoginRequestBody
}

type LoginRequestBody struct {
	Email    string `json:"email" minLength:"1"`
	Password string `json:"password" minLength:"1"` // #nosec G117 -- intentional: login request body, not a leaked secret
}

func (h *AuthHandler) Login(ctx context.Context, input *LoginInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)
	w := middleware.GetResponseWriter(ctx)

	if !h.rateLimiter.Allow(r.RemoteAddr, r) {
		return nil, errlib.NewErrorWithDetail(fmt.Errorf("rate limit exceeded for login"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
	}

	tokenPair, userID, err := h.login(ctx, &input.Body, r)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "login",
			Service:   "lugia",
			UserID:    userID,
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})
		return nil, errlib.NewErrorWithDetail(err, http.StatusUnauthorized, err.Error())
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "login",
		Service:   "lugia",
		UserID:    userID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	return nil, nil
}

func (h *AuthHandler) login(ctx context.Context, req *LoginRequestBody, r *http.Request) (*jwt.TokenPair, string, error) {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == "pending_verification" {
		return nil, user.ID.String(), fmt.Errorf("アカウントが有効化されていません。招待メールを確認し、登録を完了してください。")
	}

	if user.Status == "suspended" {
		return nil, user.ID.String(), fmt.Errorf("アカウントが停止されています。サポートにお問い合わせください。")
	}

	tenant, err := h.queries.GetTenantByID(ctx, user.TenantID)
	if err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.AuthMethod == "sso" {
		h.insertLoginAuditLog(ctx, r, tenant, user, auditlog.OutcomeFailure, "sso_only")
		return nil, user.ID.String(), fmt.Errorf("このアカウントはSSO専用です。SSOでログインしてください。")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.insertLoginAuditLog(ctx, r, tenant, user, auditlog.OutcomeFailure, "invalid_password")
		return nil, user.ID.String(), fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("login: failed to rollback transaction: %w", rbErr))
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

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.AuthJWTSecret))
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

	if err := h.insertLoginAuditLogTx(ctx, r, qtx, tenant, user, auditlog.OutcomeSuccess, ""); err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to insert audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, user.ID.String(), fmt.Errorf("failed to commit transaction: %w", err)
	}
	return tokenPair, user.ID.String(), nil
}

// insertLoginAuditLog inserts an audit log for login events outside a transaction.
// Used for failure paths where the tenant is already loaded.
func (h *AuthHandler) insertLoginAuditLog(ctx context.Context, r *http.Request, tenant *queries.Tenant, user *queries.User, outcome auditlog.Outcome, reason string) {
	var ef jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &ef); err != nil || !ef.AuditLog.Enabled {
		return
	}

	metadata := map[string]string{
		"actor_name":  user.Name,
		"actor_email": user.Email,
	}
	if reason != "" {
		metadata["reason"] = reason
	}
	metadataJSON, _ := json.Marshal(metadata)

	ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
	//nolint:auditcheck // login already failed, user gets 401 regardless — audit log is best-effort for failure events
	err := h.queries.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
		TenantID:     tenant.ID,
		ActorID:      user.ID,
		ResourceType: string(auditlog.ResourceAuth),
		Action:       string(auditlog.ActionLogin),
		Outcome:      string(outcome),
		ResourceID:   pgtype.Text{},
		Metadata:     metadataJSON,
		IpAddress:    &ipAddr,
		UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
	})
	if err != nil {
		errlib.LogError(fmt.Errorf("login: failed to insert audit log: %w", err))
	}
}

// insertLoginAuditLogTx inserts an audit log for login events within a transaction.
// Used for the success path where the audit log should be part of the same transaction.
func (h *AuthHandler) insertLoginAuditLogTx(ctx context.Context, r *http.Request, qtx *queries.Queries, tenant *queries.Tenant, user *queries.User, outcome auditlog.Outcome, reason string) error {
	var ef jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &ef); err != nil || !ef.AuditLog.Enabled {
		return nil
	}

	metadata := map[string]string{
		"actor_name":  user.Name,
		"actor_email": user.Email,
	}
	if reason != "" {
		metadata["reason"] = reason
	}
	metadataJSON, _ := json.Marshal(metadata)

	ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
	return qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
		TenantID:     tenant.ID,
		ActorID:      user.ID,
		ResourceType: string(auditlog.ResourceAuth),
		Action:       string(auditlog.ActionLogin),
		Outcome:      string(outcome),
		ResourceID:   pgtype.Text{},
		Metadata:     metadataJSON,
		IpAddress:    &ipAddr,
		UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
	})
}
