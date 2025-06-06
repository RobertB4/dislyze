package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lugia/lib/config"
	"lugia/lib/errlib"
	"lugia/lib/jwt"
	"lugia/lib/ratelimit"
	"lugia/lib/responder"
	"lugia/lib/utils"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sendgrid/sendgrid-go"
)

type SignupRequest struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type AuthHandler struct {
	dbConn      *pgxpool.Pool
	env         *config.Env
	rateLimiter *ratelimit.RateLimiter
	queries     *queries.Queries
}

func NewAuthHandler(dbConn *pgxpool.Pool, env *config.Env, rateLimiter *ratelimit.RateLimiter, queries *queries.Queries) *AuthHandler {
	return &AuthHandler{
		dbConn:      dbConn,
		env:         env,
		rateLimiter: rateLimiter,
		queries:     queries,
	}
}

func (r *SignupRequest) Validate() error {
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

	if r.CompanyName == "" {
		return fmt.Errorf("company name is required")
	}
	if r.UserName == "" {
		return fmt.Errorf("user name is required")
	}
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if r.Password != r.PasswordConfirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func (h *AuthHandler) signup(ctx context.Context, req *SignupRequest, r *http.Request) (*jwt.TokenPair, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && !errlib.Is(rErr, pgx.ErrTxClosed) {
			errlib.LogError(fmt.Errorf("failed to rollback transaction in signup: %w", rErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	tenant, err := qtx.CreateTenant(ctx, &queries.CreateTenantParams{
		Name: req.CompanyName,
		Plan: "basic",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	user, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.UserName,
		Role:         "admin",
		Status:       "active",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, user.Role, []byte(h.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tokenPair, nil
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for signup"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req SignupRequest
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

	exists, err := h.queries.ExistsUserWithEmail(r.Context(), req.Email)
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	if exists {
		appErr := errlib.New(fmt.Errorf("user already exists with this email"), http.StatusBadRequest, "このメールアドレスは既に使用されています。")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenPair, err := h.signup(r.Context(), &req, r)
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	w.WriteHeader(http.StatusOK)
}

func (r *LoginRequest) Validate() error {
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

func (h *AuthHandler) login(ctx context.Context, req *LoginRequest, r *http.Request) (*jwt.TokenPair, error) {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == "pending_verification" {
		return nil, fmt.Errorf("アカウントが有効化されていません。招待メールを確認し、登録を完了してください。")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	if user.Status == "suspended" {
		return nil, fmt.Errorf("アカウントが停止されています。サポートにお問い合わせください。")
	}

	tenant, err := h.queries.GetTenantByID(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("login: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	existingToken, err := qtx.GetRefreshTokenByUserID(ctx, user.ID)
	if err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing refresh token: %w", err)
	}

	if !errlib.Is(err, pgx.ErrNoRows) {
		err = qtx.UpdateRefreshTokenUsed(ctx, existingToken.Jti)
		if err != nil {
			return nil, fmt.Errorf("failed to update refresh token last used: %w", err)
		}
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, user.Role, []byte(h.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tokenPair, nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for login"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req LoginRequest
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

	tokenPair, err := h.login(r.Context(), &req, r)
	if err != nil {
		appErr := errlib.New(err, http.StatusUnauthorized, err.Error())
		responder.RespondWithError(w, appErr)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}

type AcceptInviteRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func (r *AcceptInviteRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)

	if r.Token == "" {
		return fmt.Errorf("token is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if r.Password != r.PasswordConfirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("AcceptInvite: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	hash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("AcceptInvite: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.queries.WithTx(tx)

	invitationTokenRecord, err := qtx.GetInvitationByTokenHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("AcceptInvite: token not found or expired for hash %s: %w", hashedTokenStr, err), http.StatusBadRequest, "招待リンクが無効か、期限切れです。お手数ですが、招待者に再度依頼してください。")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("AcceptInvite: GetInvitationByTokenHash failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	dbUser, err := qtx.GetUserByID(ctx, invitationTokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("AcceptInvite: user for valid token not found, userID: %s: %w", invitationTokenRecord.UserID, err), http.StatusInternalServerError, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("AcceptInvite: GetUserByID failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if dbUser.Status != "pending_verification" {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: user %s status is '%s', expected 'pending_verification' for token %s", dbUser.ID.String(), dbUser.Status, hashedTokenStr), http.StatusBadRequest, "このユーザーはすでに承諾済みです。")
		responder.RespondWithError(w, appErr)
		return
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to hash new password: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err = qtx.ActivateInvitedUser(ctx, &queries.ActivateInvitedUserParams{
		PasswordHash: string(hashedNewPassword),
		ID:           invitationTokenRecord.UserID,
	})
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: ActivateInvitedUser failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err = qtx.MarkInvitationTokenAsUsed(ctx, invitationTokenRecord.ID)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to mark invitation token as used ID %s: %w", invitationTokenRecord.ID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenPair, err := jwt.GenerateTokenPair(dbUser.ID, invitationTokenRecord.TenantID, dbUser.Role, []byte(h.env.JWTSecret))
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to generate token pair: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     dbUser.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to store refresh token: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	if err := tx.Commit(ctx); err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (r *ForgotPasswordRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("invalid email address format")
	}
	return nil
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.rateLimiter.Allow(r.RemoteAddr) {
		internalErr := errlib.New(fmt.Errorf("rate limit exceeded for forgot password: %s", r.RemoteAddr), http.StatusTooManyRequests, "Rate limit for forgot password")
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode forgot password request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Forgot password validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			log.Printf("ForgotPassword: No user found for email %s", req.Email)
		} else {
			internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to get user by email %s", req.Email))
			errlib.LogError(internalErr)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	resetTokenUUID, err := utils.NewUUID()
	if err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, "ForgotPassword: Failed to generate reset token UUID")
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}
	resetToken := resetTokenUUID.String()

	tokenHash := sha256.Sum256([]byte(resetToken))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, txErr := h.dbConn.Begin(ctx)
	if txErr != nil {
		internalErr := errlib.New(txErr, http.StatusInternalServerError, "ForgotPassword: Failed to begin transaction")
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ForgotPassword: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	if err := qtx.DeletePasswordResetTokenByUserID(ctx, user.ID); err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to delete existing password reset token for user %s", user.ID))
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	_, createErr := qtx.CreatePasswordResetToken(ctx, &queries.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if createErr != nil {
		internalErr := errlib.New(createErr, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to create password reset token for user %s", user.ID))
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		internalErr := errlib.New(commitErr, http.StatusInternalServerError, "ForgotPassword: Failed to commit transaction")
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", h.env.FrontendURL, resetToken)

	subject := "パスワードリセットのご案内 - dislyze"
	plainTextContent := fmt.Sprintf("%s様\n\ndislyzeアカウントのパスワードリセットリクエストを受け付けました。\n\n以下のリンクをクリックして、パスワードを再設定してください。このリンクは30分間有効です。\n%s\n\nこのメールにお心当たりがない場合は、無視してください。",
		user.Name, resetLink)
	htmlContent := fmt.Sprintf("<p>%s様</p>\n<p>dislyzeアカウントのパスワードリセットリクエストを受け付けました。</p>\n<p>以下のリンクをクリックして、パスワードを再設定してください。このリンクは30分間有効です。</p>\n<p><a href=\"%s\">パスワードを再設定する</a></p>\n<p>このメールにお心当たりがない場合は、無視してください。</p>",
		user.Name, resetLink)

	sgMailBody := SendGridMailRequestBody{
		Personalizations: []SendGridPersonalization{
			{
				To:      []SendGridEmailAddress{{Email: req.Email, Name: user.Name}},
				Subject: subject,
			},
		},
		From:    SendGridEmailAddress{Email: sendGridFromEmail, Name: sendGridFromName},
		Content: []SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: failed to marshal SendGrid request body for %s", req.Email))
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: SendGrid API call failed for %s", req.Email))
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		internalErr := errlib.New(fmt.Errorf("SendGrid API returned error status code: %d, Body: %s", sgResponse.StatusCode, sgResponse.Body), http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: SendGrid API error for %s", req.Email))
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Password reset email successfully sent via SendGrid to user with id: %s", user.ID)

	w.WriteHeader(http.StatusOK)
}

type VerifyResetTokenRequest struct {
	Token string `json:"token"`
}

func (r *VerifyResetTokenRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	if r.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (h *AuthHandler) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyResetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode verify reset token request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Verify reset token validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			internalErr := errlib.New(err, http.StatusBadRequest, fmt.Sprintf("VerifyResetToken: Token hash not found: %s", hashedTokenStr))
			errlib.LogError(internalErr)
			responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
			return
		}
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to query password reset token by hash %s", hashedTokenStr))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	if tokenRecord.UsedAt.Valid {
		internalErr := errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest, "Token already used")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		internalErr := errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest, "Token expired")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	user, err := h.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: User ID %s for valid token %s not found", tokenRecord.UserID, tokenRecord.ID))
			errlib.LogError(internalErr)
			responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
			return
		}
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to get user email for user ID %s", tokenRecord.UserID))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, map[string]string{"email": user.Email})
}

type ResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func (r *ResetPasswordRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

	if r.Token == "" {
		return fmt.Errorf("token is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if r.Password != r.PasswordConfirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode reset password request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Reset password validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			internalErr := errlib.New(err, http.StatusBadRequest, fmt.Sprintf("ResetPassword: Token hash not found: %s", hashedTokenStr))
			errlib.LogError(internalErr)
			responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
			return
		}
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to query password reset token by hash %s", hashedTokenStr))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	if tokenRecord.UsedAt.Valid {
		internalErr := errlib.New(fmt.Errorf("ResetPassword: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest, "Token already used")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		internalErr := errlib.New(fmt.Errorf("ResetPassword: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest, "Token expired")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to hash new password")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to begin transaction")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ResetPassword: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	if err := qtx.UpdateUserPassword(ctx, &queries.UpdateUserPasswordParams{
		ID:           tokenRecord.UserID,
		PasswordHash: string(hashedNewPassword),
	}); err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to update password for user ID %s", tokenRecord.UserID))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	if err := qtx.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID); err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to mark reset token ID %s as used", tokenRecord.ID))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, tokenRecord.UserID); err != nil {
		errlib.LogError(errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to delete refresh tokens for user ID %s, but password reset was successful", tokenRecord.UserID)))
	}

	if err := tx.Commit(ctx); err != nil {
		internalErr := errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to commit transaction")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	w.WriteHeader(http.StatusOK)
}
