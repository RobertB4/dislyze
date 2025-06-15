package tenants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GenerateTenantInvitationTokenRequest struct {
	Email string `json:"email"`
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate email is not empty
	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create response URL
	url := fmt.Sprintf("%s/signup?token=%s", h.env.FrontendURL, tokenString)
	response := GenerateTenantInvitationTokenResponse{
		URL: url,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}