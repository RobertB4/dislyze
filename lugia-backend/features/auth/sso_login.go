package auth

import (
	"context"
	"crypto"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/logger"
	"dislyze/jirachi/responder"
	"lugia/queries"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/jackc/pgx/v5/pgtype"
)

type SSOLoginRequest struct {
	Email string `json:"email"`
}

func (r *SSOLoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("email is invalid")
	}

	return nil
}

type SSOLoginResponse struct {
	HTML string `json:"html"`
}

func (h *AuthHandler) SSOLogin(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr, r) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for SSO login"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req SSOLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "Invalid request body")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := req.Validate(); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, err.Error())
		responder.RespondWithError(w, appErr)
		return
	}

	response, err := h.ssoLogin(r.Context(), &req)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "sso_login_initiate",
			Service:   "lugia",
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})

		appErr := errlib.New(err, http.StatusUnauthorized, "")
		responder.RespondWithError(w, appErr)
		return
	}

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "sso_login_initiate",
		Service:   "lugia",
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) ssoLogin(ctx context.Context, req *SSOLoginRequest) (*SSOLoginResponse, error) {
	parts := strings.Split(req.Email, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email format")
	}
	domain := parts[1]

	tenant, err := h.queries.GetSSOTenantByDomain(ctx, []byte(domain))
	if err != nil {
		if errlib.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no tenants for SSO domain %s found.", domain)
		}
		return nil, fmt.Errorf("failed to find tenant by domain %s: %w", domain, err)
	}

	var enterpriseFeatures authz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		return nil, fmt.Errorf("failed to parse enterprise features: %w", err)
	}

	if !enterpriseFeatures.SSO.Enabled {
		return nil, fmt.Errorf("tenant does not have SSO enabled. tenantID: %s ", tenant.ID)
	}

	keyBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderPrivateKey))
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode SP private key")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP private key: %w", err)
	}

	certBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderCertificate))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode SP certificate")
	}
	spCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP certificate: %w", err)
	}

	var idpMetadata *saml.EntityDescriptor
	metadataURL, err := url.Parse(enterpriseFeatures.SSO.IdpMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid IDP metadata URL: %w", err)
	}
	idpMetadata, err = samlsp.FetchMetadata(ctx, http.DefaultClient, *metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IDP metadata: %w", err)
	}

	acsURL, _ := url.Parse(h.env.FrontendURL + "/api/auth/sso/acs")
	spMetadataURL, _ := url.Parse(h.env.FrontendURL + "/api/auth/sso/metadata")

	sp := &saml.ServiceProvider{
		Key:               privateKey.(crypto.Signer),
		Certificate:       spCert,
		MetadataURL:       *spMetadataURL,
		AcsURL:            *acsURL,
		IDPMetadata:       idpMetadata,
		EntityID:          h.env.FrontendURL,
		AuthnNameIDFormat: saml.PersistentNameIDFormat,
		SignatureMethod:   "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
	}

	authnRequest, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPPostBinding), saml.HTTPPostBinding, saml.HTTPPostBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to create SAML request: %w", err)
	}

	authnRequest.Subject = &saml.Subject{
		NameID: &saml.NameID{
			Value: req.Email,
		},
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	err = h.queries.CreateSSOAuthRequest(ctx, &queries.CreateSSOAuthRequestParams{
		RequestID: authnRequest.ID,
		TenantID:  tenant.ID,
		Email:     req.Email,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store SSO auth request: %w", err)
	}

	authReqHTML := authnRequest.Post("")

	return &SSOLoginResponse{
		HTML: string(authReqHTML),
	}, nil
}
