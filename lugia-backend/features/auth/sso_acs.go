package auth

import (
	"context"
	"crypto"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"dislyze/jirachi/responder"
	"lugia/queries"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *AuthHandler) SSOACS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "Invalid form data")
		responder.RespondWithError(w, appErr)
		return
	}

	samlResponseBase64 := r.FormValue("SAMLResponse")
	if samlResponseBase64 == "" {
		appErr := errlib.New(fmt.Errorf("missing SAMLResponse"), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	callbackResponse, err, userErrorMessage := h.handleSSOCallback(r.Context(), samlResponseBase64, r)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "sso_acs_callback",
			Service:   "lugia",
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})

		errorMessage := userErrorMessage
		if userErrorMessage == "" {
			errorMessage = "ログインに失敗しました。管理者にお問い合わせください。"
		}

		errlib.LogError(errlib.New(err, http.StatusUnauthorized, err.Error()))
		http.Redirect(w, r, h.env.FrontendURL+"/auth/sso/login?error="+errorMessage, http.StatusFound)
		return
	}

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "sso_acs_callback",
		Service:   "lugia",
		UserID:    callbackResponse.UserID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    callbackResponse.TokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(callbackResponse.TokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    callbackResponse.TokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	http.Redirect(w, r, h.env.FrontendURL, http.StatusFound)
}

type SSOCallbackResponse struct {
	UserID    string
	TokenPair *jwt.TokenPair
}

func (h *AuthHandler) handleSSOCallback(ctx context.Context, samlResponseBase64 string, r *http.Request) (*SSOCallbackResponse, error, string) {
	samlResponseXML, err := base64.StdEncoding.DecodeString(samlResponseBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid SAML response encoding: %w", err), ""
	}

	var samlResponse saml.Response
	if err := xml.Unmarshal(samlResponseXML, &samlResponse); err != nil {
		return nil, fmt.Errorf("failed to parse SAML response: %w", err), ""
	}

	requestID := samlResponse.InResponseTo

	ssoRequest, err := h.queries.DeleteSSORequestReturning(ctx, requestID)
	if err != nil {
		logger.LogAuthEvent(logger.AuthEvent{
			EventType: "sso_invalid_in_response_to",
			Service:   "lugia",
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		})
		return nil, fmt.Errorf("invalid or expired request. request id: %s", requestID), ""
	}

	tenant, err := h.queries.GetTenantByID(ctx, ssoRequest.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant with id %s: %w", ssoRequest.TenantID, err), ""
	}

	var enterpriseFeatures authz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		return nil, fmt.Errorf("failed to parse enterprise features: %w", err), ""
	}

	if !enterpriseFeatures.SSO.Enabled {
		return nil, fmt.Errorf("SSO not enabled for tenant with id %s", tenant.ID), ""
	}

	keyBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderPrivateKey))
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode SP private key"), ""
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP private key: %w", err), ""
	}

	certBlock, _ := pem.Decode([]byte(h.env.SAMLServiceProviderCertificate))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode SP certificate"), ""
	}
	spCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP certificate: %w", err), ""
	}

	metadataURL, err := url.Parse(enterpriseFeatures.SSO.IdpMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid IDP metadata URL: %w", err), ""
	}
	idpMetadata, err := samlsp.FetchMetadata(ctx, http.DefaultClient, *metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IDP metadata: %w", err), ""
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

	assertion, err := sp.ParseResponse(r, []string{requestID})
	if err != nil {
		return nil, fmt.Errorf("failed to validate SAML response: %w", err), ""
	}

	externalSSOID := assertion.Subject.NameID.Value
	email := extractAttribute(assertion.AttributeStatements, enterpriseFeatures.SSO.AttributeMapping["email"])
	firstName := extractAttribute(assertion.AttributeStatements, enterpriseFeatures.SSO.AttributeMapping["firstName"])
	lastName := extractAttribute(assertion.AttributeStatements, enterpriseFeatures.SSO.AttributeMapping["lastName"])

	if email == "" {
		return nil, fmt.Errorf("required SAML attribute email missing"), ""
	}

	if externalSSOID == "" {
		return nil, fmt.Errorf("required SAML attribute nameID missing"), ""
	}

	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		return nil, fmt.Errorf("invalid email format: %s", email), ""
	}
	domain := emailParts[1]

	if !slices.Contains(enterpriseFeatures.SSO.AllowedDomains, domain) {
		return nil, fmt.Errorf("email domain not authorized for SSO: %s", domain), ""
	}

	user, err := h.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if !errlib.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get user: %w", err), ""
		}

		// User doesn't exist - create new user

		viewerRole, err := h.queries.GetDefaultViewerRole(ctx, ssoRequest.TenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to get default viewer role for tenant_id %s: %w", ssoRequest.TenantID, err), ""
		}

		tx, err := h.dbConn.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to start transaction: %w", err), ""
		}
		defer func() {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, sql.ErrTxDone) {
				errlib.LogError(fmt.Errorf("sso_acs: failed to rollback transaction: %w", rbErr))
			}
		}()

		qtx := h.queries.WithTx(tx)

		fullName := strings.TrimSpace(lastName + " " + firstName)
		if fullName == "" {
			fullName = email
		}

		user, err = qtx.CreateUser(ctx, &queries.CreateUserParams{
			TenantID:       ssoRequest.TenantID,
			Email:          email,
			PasswordHash:   "!",
			Name:           fullName,
			Status:         "active",
			IsInternalUser: false,
			AuthMethod:     "sso",
			ExternalSsoID:  pgtype.Text{String: externalSSOID, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err), ""
		}

		err = qtx.AssignRoleToUser(ctx, &queries.AssignRoleToUserParams{
			UserID:   user.ID,
			RoleID:   viewerRole.ID,
			TenantID: ssoRequest.TenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assign default role: %w", err), ""
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err), ""
		}

	} else {
		// User exists

		if user.TenantID != ssoRequest.TenantID {
			return nil, fmt.Errorf("user belongs to different tenant"), ""
		}

		if user.Status == "pending_verification" {
			err = h.queries.UpdateUserStatus(ctx, &queries.UpdateUserStatusParams{
				Status: "active",
				ID:     user.ID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to activate SSO user: %w", err), ""
			}
		}

		if user.Status == "suspended" {
			return nil, fmt.Errorf("account suspended"), "アカウントが停止されています。サポートにお問い合わせください。"
		}

		if user.AuthMethod == "password" {
			return nil, fmt.Errorf("user with auth_method password attempted sso login"), "このアカウントはSSOが無効です。パスワードでログインしてください。"
		}

		if !user.ExternalSsoID.Valid || user.ExternalSsoID.String == "" {
			err = h.queries.UpdateUserExternalSSOID(ctx, &queries.UpdateUserExternalSSOIDParams{
				ExternalSsoID: pgtype.Text{String: externalSSOID, Valid: true},
				ID:            user.ID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update external SSO ID for user_id %s : %w", user.ID, err), ""
			}
		}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err), ""
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("sso_acs: failed to rollback token transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	existingToken, err := qtx.GetRefreshTokenByUserID(ctx, user.ID)
	if err != nil && !errlib.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing refresh token for user_id %s: %w", user.ID, err), ""
	}

	if !errlib.Is(err, sql.ErrNoRows) {
		err = qtx.UpdateRefreshTokenUsed(ctx, existingToken.Jti)
		if err != nil {
			return nil, fmt.Errorf("failed to update refresh token used: %w", err), ""
		}
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.AuthJWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err), ""
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     user.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token for user_id %s: %w", user.ID, err), ""
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err), ""
	}

	go func() {
		cleanupCtx := context.Background()
		if err := h.queries.DeleteExpiredSSORequests(cleanupCtx); err != nil {
			errlib.LogError(fmt.Errorf("failed to delete expired SSO requests: %w", err))
		}
	}()

	return &SSOCallbackResponse{
		UserID:    user.ID.String(),
		TokenPair: tokenPair,
	}, nil, ""
}

func extractAttribute(statements []saml.AttributeStatement, attributeName string) string {
	for _, stmt := range statements {
		for _, attr := range stmt.Attributes {
			if attr.Name == attributeName {
				if len(attr.Values) > 0 {
					return attr.Values[0].Value
				}
			}
		}
	}
	return ""
}
