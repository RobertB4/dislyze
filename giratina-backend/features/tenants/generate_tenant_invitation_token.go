package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type GenerateTenantInvitationTokenRequest struct {
	Email string `json:"email"`
}

func (r *GenerateTenantInvitationTokenRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("valid email is required")
	}
	return nil
}

type GenerateTenantInvitationTokenResponse struct {
	URL string `json:"url"`
}

type TenantInvitationClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (h *TenantsHandler) GenerateTenantInvitationToken(w http.ResponseWriter, r *http.Request) {
	var req GenerateTenantInvitationTokenRequest

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

	response, err := h.generateTenantInvitationToken(r.Context(), &req)
	if err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, err.Error())
		responder.RespondWithError(w, appErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
}

func (h *TenantsHandler) generateTenantInvitationToken(ctx context.Context, req *GenerateTenantInvitationTokenRequest) (*GenerateTenantInvitationTokenResponse, error) {
	// Check if user already exists with this email
	_, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errlib.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to check user existence: %w", err)
		}
		// ErrNoRows means user doesn't exist, which is what we want - continue
	} else {
		// User exists with this email
		return nil, fmt.Errorf("このメールアドレスは既に使用されています。")
	}

	// Create JWT claims with email and 48 hour expiration
	now := time.Now()
	claims := TenantInvitationClaims{
		Email: req.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)),
		},
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.env.CreateTenantJwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT token: %w", err)
	}

	// Create response URL
	url := fmt.Sprintf("%s/signup?token=%s", h.env.FrontendURL, tokenString)
	response := &GenerateTenantInvitationTokenResponse{
		URL: url,
	}

	return response, nil
}