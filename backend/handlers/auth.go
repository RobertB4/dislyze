package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"lugia/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCompanyNameRequired = errors.New("company name is required")
	ErrUserNameRequired    = errors.New("user name is required")
	ErrEmailRequired       = errors.New("email is required")
	ErrPasswordRequired    = errors.New("password is required")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters long")
	ErrPasswordsDoNotMatch = errors.New("passwords do not match")
	ErrUserAlreadyExists   = errors.New("user with this email already exists")
)

type SignupRequest struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

type SignupResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error,omitempty"`
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

func Signup(dbConn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

		// TODO: Create tenant
		// TODO: Create user
		// TODO: Generate real JWT tokens

		// For now, return dummy tokens
		response := SignupResponse{
			AccessToken:  "dummy_access_token",
			RefreshToken: "dummy_refresh_token",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
