// Feature doc: docs/features/tenant-onboarding.md
package tenants

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/errlib"

	"github.com/golang-jwt/jwt/v5"
)

type SSOConfig struct {
	Enabled        bool     `json:"enabled"`
	IdpMetadataURL string   `json:"idp_metadata_url"`
	AllowedDomains []string `json:"allowed_domains"`
}

type GenerateTenantInvitationTokenRequest struct {
	Email       string     `json:"email"`
	CompanyName string     `json:"company_name"`
	UserName    string     `json:"user_name"`
	SSO         *SSOConfig `json:"sso,omitempty"`
}

func (r *GenerateTenantInvitationTokenRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("valid email is required")
	}

	if r.SSO != nil && r.SSO.Enabled {
		r.SSO.IdpMetadataURL = strings.TrimSpace(r.SSO.IdpMetadataURL)

		if r.SSO.IdpMetadataURL == "" {
			return fmt.Errorf("idp_metadata_url is required when SSO is enabled")
		}

		if _, err := url.ParseRequestURI(r.SSO.IdpMetadataURL); err != nil {
			return fmt.Errorf("idp_metadata_url must be a valid URL")
		}

		if len(r.SSO.AllowedDomains) == 0 {
			return fmt.Errorf("at least one allowed domain is required when SSO is enabled")
		}

		for i, domain := range r.SSO.AllowedDomains {
			r.SSO.AllowedDomains[i] = strings.TrimSpace(domain)
			if r.SSO.AllowedDomains[i] == "" {
				return fmt.Errorf("allowed domains cannot be empty")
			}
		}
	}

	return nil
}

type GenerateTenantInvitationTokenResponse struct {
	URL string `json:"url"`
}

type TenantInvitationClaims struct {
	Email       string     `json:"email"`
	CompanyName string     `json:"company_name"`
	UserName    string     `json:"user_name"`
	SSO         *SSOConfig `json:"sso,omitempty"`
	jwt.RegisteredClaims
}

// TODO: Remove SkipValidateBody once request struct fields have correct
// omitempty tags (company_name, user_name are optional but huma marks them
// required). See PROGRESS.md "SkipValidateBody workarounds".
var GenerateTokenOp = huma.Operation{
	OperationID:      "generate-tenant-invitation-token",
	Method:           http.MethodPost,
	Path:             "/tenants/generate-token",
	SkipValidateBody: true,
}

type GenerateTokenInput struct {
	Body GenerateTenantInvitationTokenRequest
}

type GenerateTokenOutput struct {
	Body GenerateTenantInvitationTokenResponse
}

func (h *TenantsHandler) GenerateTenantInvitationToken(ctx context.Context, input *GenerateTokenInput) (*GenerateTokenOutput, error) {
	if err := input.Body.Validate(); err != nil {
		return nil, errlib.NewError(fmt.Errorf("tenant invitation validation failed: %w", err), http.StatusBadRequest)
	}

	response, err := h.generateTenantInvitationToken(ctx, &input.Body)
	if err != nil {
		return nil, err
	}

	return &GenerateTokenOutput{Body: *response}, nil
}

func (h *TenantsHandler) generateTenantInvitationToken(ctx context.Context, req *GenerateTenantInvitationTokenRequest) (*GenerateTenantInvitationTokenResponse, error) {
	_, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.NewError(fmt.Errorf("failed to check user existence: %w", err), http.StatusInternalServerError)
		}
		// ErrNoRows means user doesn't exist, which is what we want - continue
	} else {
		return nil, errlib.NewErrorWithDetail(fmt.Errorf("GenerateTenantInvitationToken: email already in use"), http.StatusBadRequest, "このメールアドレスは既に使用されています。")
	}

	now := time.Now()
	claims := TenantInvitationClaims{
		Email:       req.Email,
		CompanyName: req.CompanyName,
		UserName:    req.UserName,
		SSO:         req.SSO,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.env.CreateTenantJwtSecret))
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("failed to sign JWT token: %w", err), http.StatusInternalServerError)
	}

	inviteURL := fmt.Sprintf("%s/auth/tenant-signup?token=%s", h.env.LugiaFrontendUrl, tokenString)
	response := &GenerateTenantInvitationTokenResponse{
		URL: inviteURL,
	}

	return response, nil
}
