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

	"dislyze/lib/config"
	"dislyze/lib/errors"
	"dislyze/lib/jwt"
	"dislyze/lib/ratelimit"
	"dislyze/lib/utils"
	"dislyze/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sendgrid/sendgrid-go"
)

var (
	ErrCompanyNameRequired = fmt.Errorf("会社名は必須です")
	ErrUserNameRequired    = fmt.Errorf("ユーザー名は必須です")
	ErrEmailRequired       = fmt.Errorf("メールアドレスは必須です")
	ErrPasswordRequired    = fmt.Errorf("パスワードは必須です")
	ErrPasswordTooShort    = fmt.Errorf("パスワードは8文字以上である必要があります")
	ErrPasswordsDoNotMatch = fmt.Errorf("パスワードが一致しません")
	ErrUserAlreadyExists   = fmt.Errorf("このメールアドレスは既に登録されています")
)

type SignupRequest struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

type SignupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Success bool `json:"success"`
}

type RefreshTokenInfo struct {
	ID         string    `json:"id"`
	DeviceInfo string    `json:"device_info"`
	IPAddress  string    `json:"ip_address"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsRevoked  bool      `json:"is_revoked"`
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
		return ErrCompanyNameRequired
	}
	if r.UserName == "" {
		return ErrUserNameRequired
	}
	if r.Email == "" {
		return ErrEmailRequired
	}
	if r.Password == "" {
		return ErrPasswordRequired
	}
	if len(r.Password) < 8 {
		return ErrPasswordTooShort
	}
	if r.Password != r.PasswordConfirm {
		return ErrPasswordsDoNotMatch
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
	defer tx.Rollback(ctx)

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
		appErr := errors.New(fmt.Errorf("rate limit exceeded for signup"), "Too many requests", http.StatusTooManyRequests)
		errors.LogError(appErr)
		http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
		return
	}

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode request body", http.StatusBadRequest)
		errors.LogError(appErr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		response := SignupResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	exists, err := h.queries.ExistsUserWithEmail(r.Context(), req.Email)
	if err != nil {
		appErr := errors.New(err, "Failed to check if user exists", http.StatusInternalServerError)
		errors.LogError(appErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		response := SignupResponse{Error: ErrUserAlreadyExists.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	tokenPair, err := h.signup(r.Context(), &req, r)
	if err != nil {
		appErr := errors.New(err, "Failed to create user", http.StatusInternalServerError)
		errors.LogError(appErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	response := SignupResponse{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)

	if r.Email == "" {
		return ErrEmailRequired
	}
	if r.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}

func (h *AuthHandler) login(ctx context.Context, req *LoginRequest, r *http.Request) (*jwt.TokenPair, error) {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	defer tx.Rollback(ctx)

	qtx := h.queries.WithTx(tx)

	existingToken, err := qtx.GetRefreshTokenByUserID(ctx, user.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing refresh token: %w", err)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		err = qtx.UpdateRefreshTokenLastUsed(ctx, existingToken.Jti)
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
		appErr := errors.New(fmt.Errorf("rate limit exceeded for login"), "Too many requests", http.StatusTooManyRequests)
		errors.LogError(appErr)
		http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode request body", http.StatusBadRequest)
		errors.LogError(appErr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		response := LoginResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	tokenPair, err := h.login(r.Context(), &req, r)
	if err != nil {
		response := LoginResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
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

	response := LoginResponse{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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
		errors.LogError(fmt.Errorf("AcceptInvite: failed to decode request: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: validation failed: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: failed to begin transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) && !errors.Is(rbErr, sql.ErrTxDone) {
			errors.LogError(fmt.Errorf("AcceptInvite: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.queries.WithTx(tx)

	invitationTokenRecord, err := qtx.GetInvitationByTokenHash(ctx, hashedTokenStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(fmt.Errorf("AcceptInvite: token not found or expired for hash %s", hashedTokenStr))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "招待リンクが無効か、期限切れです。お手数ですが、招待者に再度依頼してください。"})
			return
		}
		errors.LogError(fmt.Errorf("AcceptInvite: GetInvitationByTokenHash failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser, err := qtx.GetUserByID(ctx, invitationTokenRecord.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(fmt.Errorf("AcceptInvite: user for valid token not found, userID: %s", invitationTokenRecord.UserID.String()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		errors.LogError(fmt.Errorf("AcceptInvite: GetUserByID failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if dbUser.Status != "pending_verification" {
		errors.LogError(fmt.Errorf("AcceptInvite: user %s status is '%s', expected 'pending_verification' for token %s", dbUser.ID.String(), dbUser.Status, hashedTokenStr))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "このユーザーはすでに承諾済みです。"})
		return
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: failed to hash new password: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = qtx.ActivateInvitedUser(ctx, &queries.ActivateInvitedUserParams{
		PasswordHash: string(hashedNewPassword),
		ID:           invitationTokenRecord.UserID,
	})
	if err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: ActivateInvitedUser failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = qtx.DeleteInvitationToken(ctx, invitationTokenRecord.ID)
	if err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: failed to delete used invitation token ID %s: %w", invitationTokenRecord.ID.String(), err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenPair, err := jwt.GenerateTokenPair(dbUser.ID, invitationTokenRecord.TenantID, dbUser.Role, []byte(h.env.JWTSecret))
	if err != nil {
		errors.LogError(fmt.Errorf("AcceptInvite: failed to generate token pair: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("AcceptInvite: failed to store refresh token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("AcceptInvite: failed to commit transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (r *ForgotPasswordRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		return ErrEmailRequired
	}
	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("invalid email address format")
	}
	return nil
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.rateLimiter.Allow(r.RemoteAddr) {
		errors.LogError(errors.New(fmt.Errorf("rate limit exceeded for forgot password: %s", r.RemoteAddr), "Rate limit", http.StatusTooManyRequests))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode forgot password request body", http.StatusBadRequest)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: false})
		return
	}

	if err := req.Validate(); err != nil {
		errors.LogError(errors.New(err, "Forgot password validation failed", http.StatusBadRequest))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: false})
		return
	}

	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("ForgotPassword: No user found for email %s", req.Email)
		} else {
			appErr := errors.New(err, fmt.Sprintf("ForgotPassword: Failed to get user by email %s", req.Email), http.StatusInternalServerError)
			errors.LogError(appErr)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	resetTokenUUID, err := utils.NewUUID()
	if err != nil {
		appErr := errors.New(err, "ForgotPassword: Failed to generate reset token UUID", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}
	resetToken := resetTokenUUID.String()

	tokenHash := sha256.Sum256([]byte(resetToken))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, txErr := h.dbConn.Begin(ctx)
	if txErr != nil {
		appErr := errors.New(txErr, "ForgotPassword: Failed to begin transaction", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}
	defer tx.Rollback(ctx)

	qtx := h.queries.WithTx(tx)

	if err := qtx.DeletePasswordResetTokenByUserID(ctx, user.ID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		appErr := errors.New(err, fmt.Sprintf("ForgotPassword: Failed to delete existing password reset token for user %s", user.ID), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	_, createErr := qtx.CreatePasswordResetToken(ctx, &queries.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if createErr != nil {
		appErr := errors.New(createErr, fmt.Sprintf("ForgotPassword: Failed to create password reset token for user %s", user.ID), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		appErr := errors.New(commitErr, "ForgotPassword: Failed to commit transaction", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
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
		appErr := errors.New(err, fmt.Sprintf("ForgotPassword: failed to marshal SendGrid request body for %s", req.Email), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		appErr := errors.New(err, fmt.Sprintf("ForgotPassword: SendGrid API call failed for %s", req.Email), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		appErr := errors.New(fmt.Errorf("SendGrid API returned error status code: %d, Body: %s", sgResponse.StatusCode, sgResponse.Body), fmt.Sprintf("ForgotPassword: SendGrid API error for %s", req.Email), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
		return
	}

	log.Printf("Password reset email successfully sent via SendGrid to user with id: %s", user.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ForgotPasswordResponse{Success: true})
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

type VerifyResetTokenResponse struct {
	Success bool   `json:"success"`
	Email   string `json:"email,omitempty"`
}

func (h *AuthHandler) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyResetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode verify reset token request body", http.StatusBadRequest)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	if err := req.Validate(); err != nil {
		appErr := errors.New(err, "Verify reset token validation failed", http.StatusBadRequest)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(errors.New(err, fmt.Sprintf("VerifyResetToken: Token hash not found: %s", hashedTokenStr), http.StatusBadRequest))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
			return
		}

		appErr := errors.New(err, fmt.Sprintf("VerifyResetToken: Failed to query password reset token by hash %s", hashedTokenStr), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	if tokenRecord.UsedAt.Valid {
		errors.LogError(errors.New(fmt.Errorf("VerifyResetToken: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), "Token already used", http.StatusBadRequest))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		errors.LogError(errors.New(fmt.Errorf("VerifyResetToken: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), "Token expired", http.StatusBadRequest))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	user, err := h.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			appErr := errors.New(err, fmt.Sprintf("VerifyResetToken: User ID %s for valid token %s not found", tokenRecord.UserID, tokenRecord.ID), http.StatusInternalServerError)
			errors.LogError(appErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
			return
		}
		appErr := errors.New(err, fmt.Sprintf("VerifyResetToken: Failed to get user email for user ID %s", tokenRecord.UserID), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VerifyResetTokenResponse{Success: true, Email: user.Email})
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
		return ErrPasswordRequired
	}
	if len(r.Password) < 8 {
		return ErrPasswordTooShort
	}
	if r.Password != r.PasswordConfirm {
		return ErrPasswordsDoNotMatch
	}
	return nil
}

type ResetPasswordResponse struct {
	Success bool `json:"success"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode reset password request body", http.StatusBadRequest)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	if err := req.Validate(); err != nil {
		appErr := errors.New(err, "Reset password validation failed", http.StatusBadRequest)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(errors.New(err, fmt.Sprintf("ResetPassword: Token hash not found: %s", hashedTokenStr), http.StatusBadRequest))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
			return
		}
		appErr := errors.New(err, fmt.Sprintf("ResetPassword: Failed to query password reset token by hash %s", hashedTokenStr), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	if tokenRecord.UsedAt.Valid {
		errors.LogError(errors.New(fmt.Errorf("ResetPassword: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), "Token already used", http.StatusBadRequest))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		errors.LogError(errors.New(fmt.Errorf("ResetPassword: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), "Token expired", http.StatusBadRequest))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		appErr := errors.New(err, "ResetPassword: Failed to hash new password", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errors.New(err, "ResetPassword: Failed to begin transaction", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}
	defer tx.Rollback(ctx)

	qtx := h.queries.WithTx(tx)

	if err := qtx.UpdateUserPassword(ctx, &queries.UpdateUserPasswordParams{
		ID:           tokenRecord.UserID,
		PasswordHash: string(hashedNewPassword),
	}); err != nil {
		appErr := errors.New(err, fmt.Sprintf("ResetPassword: Failed to update password for user ID %s", tokenRecord.UserID), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	if err := qtx.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID); err != nil {
		appErr := errors.New(err, fmt.Sprintf("ResetPassword: Failed to mark reset token ID %s as used", tokenRecord.ID), http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, tokenRecord.UserID); err != nil {
		errors.LogError(errors.New(err, fmt.Sprintf("ResetPassword: Failed to delete refresh tokens for user ID %s, but password reset was successful", tokenRecord.UserID), http.StatusInternalServerError))
	}

	if err := tx.Commit(ctx); err != nil {
		appErr := errors.New(err, "ResetPassword: Failed to commit transaction", http.StatusInternalServerError)
		errors.LogError(appErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResetPasswordResponse{Success: false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResetPasswordResponse{Success: true})
}
