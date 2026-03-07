// Feature doc: docs/features/tenant-onboarding.md
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	golangJwt "github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/errlib"
	jirachijwt "dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"lugia/lib/humautil"
	"lugia/lib/middleware"
	"lugia/queries"
)

var TenantSignupOp = huma.Operation{
	OperationID: "tenant-signup",
	Method:      http.MethodPost,
	Path:        "/auth/tenant-signup",
}

type TenantSignupInput struct {
	Token string `query:"token"`
	Body  TenantSignupRequestBody
}

type TenantSignupRequestBody struct {
	Password        string `json:"password"` // #nosec G117 -- intentional: request body, not a leaked secret
	PasswordConfirm string `json:"password_confirm"`
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
}

type SSOConfig struct {
	Enabled        bool     `json:"enabled"`
	IdpMetadataURL string   `json:"idp_metadata_url"`
	AllowedDomains []string `json:"allowed_domains"`
}

type CreateTenantTokenClaims struct {
	Email       string     `json:"email"`
	CompanyName string     `json:"company_name"`
	UserName    string     `json:"user_name"`
	SSO         *SSOConfig `json:"sso,omitempty"`
	golangJwt.RegisteredClaims
}

func (r *TenantSignupRequestBody) ValidatePasswordSignup() error {
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)

	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if r.Password != r.PasswordConfirm {
		return fmt.Errorf("passwords do not match")
	}

	if r.CompanyName == "" {
		return fmt.Errorf("company_name is required")
	}

	if r.UserName == "" {
		return fmt.Errorf("user_name is required")
	}

	return nil
}

func (r *TenantSignupRequestBody) ValidateSSOSignup() error {
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)

	if r.CompanyName == "" {
		return fmt.Errorf("company_name is required")
	}

	if r.UserName == "" {
		return fmt.Errorf("user_name is required")
	}

	return nil
}

func (h *AuthHandler) TenantSignup(ctx context.Context, input *TenantSignupInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)
	w := middleware.GetResponseWriter(ctx)

	if !h.rateLimiter.Allow(r.RemoteAddr, r) {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("rate limit exceeded for tenant signup"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
	}

	tokenString := input.Token
	if tokenString == "" {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("tenant signup token is empty"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
	}

	decodedToken, urlErr := url.QueryUnescape(tokenString)
	if urlErr == nil {
		tokenString = decodedToken
	}

	token, err := golangJwt.ParseWithClaims(tokenString, &CreateTenantTokenClaims{}, func(token *golangJwt.Token) (any, error) {
		if _, ok := token.Method.(*golangJwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.env.CreateTenantJwtSecret), nil
	})
	if err != nil {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("tenant signup token parse failed: %w", err), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
	}

	claims, ok := token.Claims.(*CreateTenantTokenClaims)
	if !ok || !token.Valid {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("tenant signup token claims invalid"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
	}

	if strings.TrimSpace(claims.Email) == "" {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("tenant signup token has empty email"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
	}

	exists, err := h.queries.ExistsUserWithEmail(ctx, claims.Email)
	if err != nil {
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}
	if exists {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("tenant signup attempted with existing email"), http.StatusBadRequest, "このメールアドレスは既に使用されています。")
	}

	if claims.SSO != nil && claims.SSO.Enabled {
		if err := input.Body.ValidateSSOSignup(); err != nil {
			return nil, humautil.NewError(fmt.Errorf("tenant signup SSO validation failed: %w", err), http.StatusBadRequest)
		}

		if err := h.ssoTenantSignup(ctx, &input.Body, claims); err != nil {
			return nil, humautil.NewError(err, http.StatusInternalServerError)
		}
		return nil, nil
	}

	if err := input.Body.ValidatePasswordSignup(); err != nil {
		return nil, humautil.NewError(fmt.Errorf("tenant signup password validation failed: %w", err), http.StatusBadRequest)
	}

	tokenPair, err := h.passwordTenantSignup(ctx, &input.Body, claims, r)
	if err != nil {
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.env.IsCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	return nil, nil
}

func (h *AuthHandler) passwordTenantSignup(ctx context.Context, req *TenantSignupRequestBody, claims *CreateTenantTokenClaims, r *http.Request) (*jirachijwt.TokenPair, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	internalUserHashedPassword, err := bcrypt.GenerateFromPassword([]byte(h.env.InternalUserPW), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && !errlib.Is(rErr, pgx.ErrTxClosed) {
			errlib.LogError(fmt.Errorf("failed to rollback transaction in tenant signup: %w", rErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	tenant, err := qtx.CreateTenant(ctx, &queries.CreateTenantParams{
		Name:               claims.CompanyName,
		AuthMethod:         "password",
		EnterpriseFeatures: []byte("{}"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	user, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:       tenant.ID,
		Email:          claims.Email,
		PasswordHash:   string(hashedPassword),
		Name:           req.UserName,
		Status:         "active",
		IsInternalUser: false,
		ExternalSsoID:  pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	internalUser, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:       tenant.ID,
		Email:          fmt.Sprintf("%s@internal.com", tenant.ID),
		PasswordHash:   string(internalUserHashedPassword),
		Name:           "内部ユーザー",
		Status:         "active",
		IsInternalUser: true,
		ExternalSsoID:  pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create internal user %w", err)
	}

	adminRoleID, err := h.setupDefaultRoles(ctx, qtx, tenant.ID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup default roles: %w", err)
	}

	err = qtx.AssignRoleToUser(ctx, &queries.AssignRoleToUserParams{
		UserID:   internalUser.ID,
		RoleID:   adminRoleID,
		TenantID: tenant.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assign admin role to internal user: %w", err)
	}

	tokenPair, err := jirachijwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.AuthJWTSecret))
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

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "tenant_signup",
		Service:   "lugia",
		UserID:    user.ID.String(),
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Success:   true,
	})

	return tokenPair, nil
}

func (h *AuthHandler) ssoTenantSignup(ctx context.Context, req *TenantSignupRequestBody, claims *CreateTenantTokenClaims) error {
	internalUserHashedPassword, err := bcrypt.GenerateFromPassword([]byte(h.env.InternalUserPW), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && !errlib.Is(rErr, pgx.ErrTxClosed) {
			errlib.LogError(fmt.Errorf("failed to rollback transaction in sso tenant signup: %w", rErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	enterpriseFeaturesJSON, err := json.Marshal(map[string]any{
		"sso": map[string]any{
			"enabled":          claims.SSO.Enabled,
			"idp_metadata_url": claims.SSO.IdpMetadataURL,
			"attribute_mapping": map[string]string{
				"email":     "email",
				"firstName": "firstName",
				"lastName":  "lastName",
			},
			"allowed_domains": claims.SSO.AllowedDomains,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal enterprise features: %w", err)
	}

	tenant, err := qtx.CreateTenant(ctx, &queries.CreateTenantParams{
		Name:               req.CompanyName,
		AuthMethod:         "sso",
		EnterpriseFeatures: enterpriseFeaturesJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	user, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:       tenant.ID,
		Email:          claims.Email,
		PasswordHash:   "!",
		Name:           req.UserName,
		Status:         "active",
		IsInternalUser: false,
		ExternalSsoID:  pgtype.Text{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	internalUser, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:       tenant.ID,
		Email:          fmt.Sprintf("%s@internal.com", tenant.ID),
		PasswordHash:   string(internalUserHashedPassword),
		Name:           "内部ユーザー",
		Status:         "active",
		IsInternalUser: true,
		ExternalSsoID:  pgtype.Text{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("failed to create internal user: %w", err)
	}

	adminRoleID, err := h.setupDefaultRoles(ctx, qtx, tenant.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to setup default roles: %w", err)
	}

	err = qtx.AssignRoleToUser(ctx, &queries.AssignRoleToUserParams{
		UserID:   internalUser.ID,
		RoleID:   adminRoleID,
		TenantID: tenant.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign admin role to internal user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.LogAuthEvent(logger.AuthEvent{
		EventType: "sso_tenant_signup",
		Service:   "lugia",
		UserID:    user.ID.String(),
		IPAddress: "",
		UserAgent: "",
		Timestamp: time.Now(),
		Success:   true,
	})

	return nil
}
