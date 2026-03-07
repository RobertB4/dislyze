// Feature doc: docs/features/tenant-onboarding.md
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/jwt"
	"lugia/lib/middleware"
	"lugia/queries"
)

var AcceptInviteOp = huma.Operation{
	OperationID: "accept-invite",
	Method:      http.MethodPost,
	Path:        "/auth/accept-invite",
}

type AcceptInviteInput struct {
	Body AcceptInviteRequestBody
}

type AcceptInviteRequestBody struct {
	Token           string `json:"token"`
	Password        string `json:"password"` // #nosec G117 -- intentional: request body, not a leaked secret
	PasswordConfirm string `json:"password_confirm"`
}

func (r *AcceptInviteRequestBody) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	r.Password = strings.TrimSpace(r.Password)
	r.PasswordConfirm = strings.TrimSpace(r.PasswordConfirm)

	if r.Token == "" {
		return fmt.Errorf("token is required")
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

func (h *AuthHandler) AcceptInvite(ctx context.Context, input *AcceptInviteInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)
	w := middleware.GetResponseWriter(ctx)

	if err := input.Body.Validate(); err != nil {
		return nil, errlib.NewError(fmt.Errorf("accept invite validation failed: %w", err), http.StatusBadRequest)
	}

	tokenPair, err := h.acceptInvite(ctx, input.Body, r)
	if err != nil {
		return nil, err
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

func (h *AuthHandler) acceptInvite(ctx context.Context, req AcceptInviteRequestBody, r *http.Request) (*jwt.TokenPair, error) {
	hash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("AcceptInvite: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.queries.WithTx(tx)

	invitationTokenRecord, err := qtx.GetInvitationByTokenHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.NewErrorWithDetail(fmt.Errorf("AcceptInvite: token not found or expired for hash %s: %w", hashedTokenStr, err), http.StatusBadRequest, "招待リンクが無効か、期限切れです。お手数ですが、招待者に再度依頼してください。")
		}
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: GetInvitationByTokenHash failed: %w", err), http.StatusInternalServerError)
	}

	dbUser, err := qtx.GetUserByID(ctx, invitationTokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.NewError(fmt.Errorf("AcceptInvite: user for valid token not found, userID: %s: %w", invitationTokenRecord.UserID, err), http.StatusInternalServerError)
		}
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: GetUserByID failed: %w", err), http.StatusInternalServerError)
	}

	tenant, err := qtx.GetTenantByID(ctx, dbUser.TenantID)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to get tenant: %w", err), http.StatusInternalServerError)
	}

	if tenant.AuthMethod == "sso" {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: user belongs to SSO tenant"), http.StatusBadRequest)
	}

	if dbUser.Status != "pending_verification" {
		return nil, errlib.NewErrorWithDetail(fmt.Errorf("AcceptInvite: user %s status is '%s', expected 'pending_verification' for token %s", dbUser.ID.String(), dbUser.Status, hashedTokenStr), http.StatusBadRequest, "このユーザーはすでに承諾済みです。")
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to hash new password: %w", err), http.StatusInternalServerError)
	}

	err = qtx.ActivateInvitedUser(ctx, &queries.ActivateInvitedUserParams{
		PasswordHash: string(hashedNewPassword),
		ID:           invitationTokenRecord.UserID,
	})
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: ActivateInvitedUser failed: %w", err), http.StatusInternalServerError)
	}

	err = qtx.MarkInvitationTokenAsUsed(ctx, invitationTokenRecord.ID)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to mark invitation token as used ID %s: %w", invitationTokenRecord.ID.String(), err), http.StatusInternalServerError)
	}

	tokenPair, err := jwt.GenerateTokenPair(dbUser.ID, invitationTokenRecord.TenantID, []byte(h.env.AuthJWTSecret))
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to generate token pair: %w", err), http.StatusInternalServerError)
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     dbUser.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to store refresh token: %w", err), http.StatusInternalServerError)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errlib.NewError(fmt.Errorf("AcceptInvite: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return tokenPair, nil
}
