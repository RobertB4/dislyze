package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"dislyze/jirachi/responder"
	"giratina/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LoginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequestBody) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr, r) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for login"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req LoginRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "Invalid request body")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := req.Validate(); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenPair, userID, err := h.login(r.Context(), &req, r)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "login",
			Service:   "giratina",
			UserID:    userID,
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})
		appErr := errlib.New(err, http.StatusUnauthorized, err.Error())
		responder.RespondWithError(w, appErr)
		return
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
		Service:   "giratina",
		UserID:    userID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) login(ctx context.Context, req *LoginRequestBody, r *http.Request) (*jwt.TokenPair, string, error) {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	userID := user.ID.String()

	if !user.IsInternalAdmin {
		return nil, userID, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	if user.Status == "pending_verification" {
		return nil, userID, fmt.Errorf("アカウントが有効化されていません。招待メールを確認し、登録を完了してください。")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, userID, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	if user.Status == "suspended" {
		return nil, userID, fmt.Errorf("アカウントが停止されています。サポートにお問い合わせください。")
	}

	tenant, err := h.queries.GetTenantByID(ctx, user.TenantID)
	if err != nil {
		return nil, userID, fmt.Errorf("failed to get tenant: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, userID, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("login: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	existingToken, err := qtx.GetRefreshTokenByUserID(ctx, user.ID)
	if err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return nil, userID, fmt.Errorf("failed to check existing refresh token: %w", err)
	}

	if !errlib.Is(err, pgx.ErrNoRows) {
		err = qtx.UpdateRefreshTokenUsed(ctx, existingToken.Jti)
		if err != nil {
			return nil, userID, fmt.Errorf("failed to update refresh token last used: %w", err)
		}
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.AuthJWTSecret))
	if err != nil {
		return nil, userID, fmt.Errorf("failed to generate token pair: %w", err)
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, userID, fmt.Errorf("failed to store refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, userID, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return tokenPair, userID, nil
}
