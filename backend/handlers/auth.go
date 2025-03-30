package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lugia/lib/config"
	"lugia/lib/errors"
	"lugia/lib/jwt"
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

type AuthHandler struct {
	dbConn *pgxpool.Pool
	env    *config.Env
}

func NewAuthHandler(dbConn *pgxpool.Pool, env *config.Env) *AuthHandler {
	return &AuthHandler{
		dbConn: dbConn,
		env:    env,
	}
}

func (r *SignupRequest) Validate() error {
	// Trim whitespace from all fields
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

	// Check for empty or whitespace-only fields
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

// signup handles user registration and returns a token pair
func (h *AuthHandler) signup(ctx context.Context, req *SignupRequest, r *http.Request) (*jwt.TokenPair, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Start a transaction
	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create queries instance for transaction
	qtx := queries.New(tx)

	// Create tenant
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

	// Create user
	user, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name: pgtype.Text{
			String: req.UserName,
			Valid:  true,
		},
		Role: "admin",
		Status: pgtype.Text{
			String: "active",
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token pair
	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, user.Role, []byte(h.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Hash the refresh token before storing
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(tokenPair.RefreshToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	// Store refresh token
	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  string(hashedToken),
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tokenPair, nil
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode request body", http.StatusBadRequest)
		errors.LogError(appErr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response := SignupResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if user already exists
	exists, err := queries.New(h.dbConn).ExistsUserWithEmail(r.Context(), req.Email)
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

	// Create user and get token pair
	tokenPair, err := h.signup(r.Context(), &req, r)
	if err != nil {
		appErr := errors.New(err, "Failed to create user", http.StatusInternalServerError)
		errors.LogError(appErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set cookies
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
	// Get user by email
	fmt.Println("req.Email", req.Email)
	user, err := queries.New(h.dbConn).GetUserByEmail(ctx, req.Email)
	fmt.Println("user", user, err)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		fmt.Println("err", err)
		return nil, fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	// Get tenant
	tenant, err := queries.New(h.dbConn).GetTenantByID(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Generate token pair
	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, user.Role, []byte(h.env.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Hash the refresh token before storing
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(tokenPair.RefreshToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	// Store refresh token
	_, err = queries.New(h.dbConn).CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  string(hashedToken),
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return tokenPair, nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.New(err, "Failed to decode request body", http.StatusBadRequest)
		errors.LogError(appErr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response := LoginResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Authenticate user and get token pair
	tokenPair, err := h.login(r.Context(), &req, r)
	if err != nil {
		response := LoginResponse{Error: err.Error()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set cookies
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
