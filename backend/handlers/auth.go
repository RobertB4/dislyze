package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lugia/lib/config"
	"lugia/lib/errors"
	"lugia/lib/jwt"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
		Status: pgtype.Text{
			String: "active",
			Valid:  true,
		},
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
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
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
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
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
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// Clear refresh_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// AcceptInviteRequest defines the structure for the accept invitation request body.
type AcceptInviteRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

// Validate checks if the AcceptInviteRequest fields are valid.
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

// AcceptInvite handles the process of a user accepting an invitation.
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
			w.WriteHeader(http.StatusUnauthorized) // Token invalid or expired
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
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
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
