package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/errlib"
	jirachijwt "dislyze/jirachi/jwt"
	"dislyze/jirachi/logger"
	"dislyze/jirachi/responder"
	"lugia/queries"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TenantSignupRequestBody struct {
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
}

type CreateTenantTokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (r *TenantSignupRequestBody) Validate() error {
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

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

func (h *AuthHandler) TenantSignup(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for tenant signup"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		appErr := errlib.New(fmt.Errorf("token is required"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
		responder.RespondWithError(w, appErr)
		return
	}

	decodedToken, urlErr := url.QueryUnescape(tokenString)
	if urlErr == nil {
		tokenString = decodedToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &CreateTenantTokenClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.env.CreateTenantJwtSecret), nil
	})
	if err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "無効または期限切れの招待リンクです。")
		responder.RespondWithError(w, appErr)
		return
	}

	claims, ok := token.Claims.(*CreateTenantTokenClaims)
	if !ok || !token.Valid {
		appErr := errlib.New(fmt.Errorf("invalid token claims"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
		responder.RespondWithError(w, appErr)
		return
	}

	if strings.TrimSpace(claims.Email) == "" {
		appErr := errlib.New(fmt.Errorf("email is required in token claims"), http.StatusBadRequest, "無効または期限切れの招待リンクです。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req TenantSignupRequestBody
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

	exists, err := h.queries.ExistsUserWithEmail(r.Context(), claims.Email)
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	if exists {
		appErr := errlib.New(fmt.Errorf("user already exists with this email"), http.StatusBadRequest, "このメールアドレスは既に使用されています。")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenPair, err := h.tenantSignup(r.Context(), &req, claims, r)
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
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

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) tenantSignup(ctx context.Context, req *TenantSignupRequestBody, claims *CreateTenantTokenClaims, r *http.Request) (*jirachijwt.TokenPair, error) {
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

	tenant, err := qtx.CreateTenant(ctx, req.CompanyName)
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	_, err = qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:       tenant.ID,
		Email:          fmt.Sprintf("%s@internal.com", tenant.ID),
		PasswordHash:   string(internalUserHashedPassword),
		Name:           "内部ユーザー",
		Status:         "active",
		IsInternalUser: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create internal user %w", err)
	}

	err = h.setupDefaultRoles(ctx, qtx, tenant.ID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup default roles: %w", err)
	}

	tokenPair, err := jirachijwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.JWTSecret))
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
