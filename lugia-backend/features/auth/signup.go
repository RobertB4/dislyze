package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lugia/lib/errlib"
	"lugia/lib/jwt"
	"lugia/lib/responder"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type SignupRequestBody struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func (r *SignupRequestBody) Validate() error {
	r.CompanyName = strings.TrimSpace(r.CompanyName)
	r.UserName = strings.TrimSpace(r.UserName)
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

	if r.CompanyName == "" {
		return fmt.Errorf("company name is required")
	}
	if r.UserName == "" {
		return fmt.Errorf("user name is required")
	}
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if r.Password != r.PasswordConfirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if !h.rateLimiter.Allow(r.RemoteAddr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for signup"), http.StatusTooManyRequests, "試行回数が上限を超えました。お手数ですが、しばらく時間をおいてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var req SignupRequestBody
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

	exists, err := h.queries.ExistsUserWithEmail(r.Context(), req.Email)
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

	tokenPair, err := h.signup(r.Context(), &req, r)
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
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenPair.ExpiresIn),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "dislyze_refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) signup(ctx context.Context, req *SignupRequestBody, r *http.Request) (*jwt.TokenPair, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && !errlib.Is(rErr, pgx.ErrTxClosed) {
			errlib.LogError(fmt.Errorf("failed to rollback transaction in signup: %w", rErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	tenant, err := qtx.CreateTenant(ctx, req.CompanyName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	user, err := qtx.CreateUser(ctx, &queries.CreateUserParams{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.UserName,
		Status:       "active",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	err = h.setupDefaultRoles(ctx, qtx, tenant.ID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup default roles: %w", err)
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, tenant.ID, []byte(h.env.JWTSecret))
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

	return tokenPair, nil
}

// By default, the admin (管理者) role has all permissions
// the editor (編集者) has no permissions
func (h *AuthHandler) setupDefaultRoles(ctx context.Context, qtx *queries.Queries, tenantID pgtype.UUID, userID pgtype.UUID) error {
	permissions, err := qtx.GetAllPermissions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	adminRole, err := qtx.CreateRole(ctx, &queries.CreateRoleParams{
		TenantID:    tenantID,
		Name:        "管理者",
		Description: pgtype.Text{String: "すべての機能にアクセス可能", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create admin role: %w", err)
	}

	permissionIDs := make([]pgtype.UUID, len(permissions))
	for i, permission := range permissions {
		permissionIDs[i] = permission.ID
	}

	err = qtx.CreateRolePermissionsBulk(ctx, &queries.CreateRolePermissionsBulkParams{
		RoleID:        adminRole.ID,
		PermissionIds: permissionIDs,
		TenantID:      tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign permissions to admin role: %w", err)
	}

	_, err = qtx.CreateRole(ctx, &queries.CreateRoleParams{
		TenantID:    tenantID,
		Name:        "編集者",
		Description: pgtype.Text{String: "限定的な編集権限", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create editor role: %w", err)
	}

	err = qtx.AssignRoleToUser(ctx, &queries.AssignRoleToUserParams{
		UserID:   userID,
		RoleID:   adminRole.ID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign admin role to user: %w", err)
	}

	return nil
}
