package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lugia/config"
	"lugia/errors"
	"lugia/jwt"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCompanyNameRequired = fmt.Errorf("company name is required")
	ErrUserNameRequired    = fmt.Errorf("user name is required")
	ErrEmailRequired       = fmt.Errorf("email is required")
	ErrPasswordRequired    = fmt.Errorf("password is required")
	ErrPasswordTooShort    = fmt.Errorf("password must be at least 8 characters long")
	ErrPasswordsDoNotMatch = fmt.Errorf("passwords do not match")
	ErrUserAlreadyExists   = fmt.Errorf("user with this email already exists")
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

func Signup(dbConn *pgxpool.Pool, env *config.Env) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Initialize queries with database connection
		q := queries.New(dbConn)

		// Check if user already exists
		exists, err := q.ExistsUserWithEmail(r.Context(), req.Email)
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

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			appErr := errors.New(err, "Failed to hash password", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Start a transaction
		tx, err := dbConn.Begin(r.Context())
		if err != nil {
			appErr := errors.New(err, "Failed to start transaction", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		// Create queries instance for transaction
		qtx := queries.New(tx)

		// Create tenant
		tenant, err := qtx.CreateTenant(r.Context(), &queries.CreateTenantParams{
			Name: req.CompanyName,
			Plan: "basic",
			Status: pgtype.Text{
				String: "active",
				Valid:  true,
			},
		})
		if err != nil {
			appErr := errors.New(err, "Failed to create tenant", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Create user
		user, err := qtx.CreateUser(r.Context(), &queries.CreateUserParams{
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
			appErr := errors.New(err, "Failed to create user", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate tokens
		tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, user.Role, []byte(env.JWTSecret))
		if err != nil {
			appErr := errors.New(err, "Failed to generate tokens", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Hash refresh token for storage
		hashedRefreshToken, err := bcrypt.GenerateFromPassword([]byte(tokenPair.RefreshToken), bcrypt.DefaultCost)
		if err != nil {
			appErr := errors.New(err, "Failed to hash refresh token", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Store refresh token
		_, err = qtx.CreateRefreshToken(r.Context(), &queries.CreateRefreshTokenParams{
			UserID:     user.ID,
			TokenHash:  string(hashedRefreshToken),
			DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
			IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
			ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
		})
		if err != nil {
			appErr := errors.New(err, "Failed to store refresh token", http.StatusInternalServerError)
			errors.LogError(appErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Commit the transaction
		if err := tx.Commit(r.Context()); err != nil {
			appErr := errors.New(err, "Failed to commit transaction", http.StatusInternalServerError)
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
}
