package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
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

type AcceptInviteRequestBody struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
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

func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AcceptInviteRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("AcceptInvite: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("AcceptInvite: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.acceptInvite(ctx, req, w, r)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) acceptInvite(ctx context.Context, req AcceptInviteRequestBody, w http.ResponseWriter, r *http.Request) error {
	hash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
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
			return errlib.New(fmt.Errorf("AcceptInvite: token not found or expired for hash %s: %w", hashedTokenStr, err), http.StatusBadRequest, "招待リンクが無効か、期限切れです。お手数ですが、招待者に再度依頼してください。")
		}
		return errlib.New(fmt.Errorf("AcceptInvite: GetInvitationByTokenHash failed: %w", err), http.StatusInternalServerError, "")
	}

	dbUser, err := qtx.GetUserByID(ctx, invitationTokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("AcceptInvite: user for valid token not found, userID: %s: %w", invitationTokenRecord.UserID, err), http.StatusInternalServerError, "")

		}
		return errlib.New(fmt.Errorf("AcceptInvite: GetUserByID failed: %w", err), http.StatusInternalServerError, "")
	}

	if dbUser.Status != "pending_verification" {
		return errlib.New(fmt.Errorf("AcceptInvite: user %s status is '%s', expected 'pending_verification' for token %s", dbUser.ID.String(), dbUser.Status, hashedTokenStr), http.StatusBadRequest, "このユーザーはすでに承諾済みです。")
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to hash new password: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.ActivateInvitedUser(ctx, &queries.ActivateInvitedUserParams{
		PasswordHash: string(hashedNewPassword),
		ID:           invitationTokenRecord.UserID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: ActivateInvitedUser failed: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.MarkInvitationTokenAsUsed(ctx, invitationTokenRecord.ID)
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to mark invitation token as used ID %s: %w", invitationTokenRecord.ID.String(), err), http.StatusInternalServerError, "")
	}

	tokenPair, err := jwt.GenerateTokenPair(dbUser.ID, invitationTokenRecord.TenantID, []byte(h.env.JWTSecret))
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to generate token pair: %w", err), http.StatusInternalServerError, "")
	}

	_, err = qtx.CreateRefreshToken(ctx, &queries.CreateRefreshTokenParams{
		UserID:     dbUser.ID,
		Jti:        tokenPair.JTI,
		DeviceInfo: pgtype.Text{String: r.UserAgent(), Valid: true},
		IpAddress:  pgtype.Text{String: r.RemoteAddr, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to store refresh token: %w", err), http.StatusInternalServerError, "")
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

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("AcceptInvite: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
