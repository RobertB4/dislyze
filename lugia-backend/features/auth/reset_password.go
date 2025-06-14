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

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
)

type ResetPasswordRequestBody struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func (r *ResetPasswordRequestBody) Validate() error {
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

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResetPasswordRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode reset password request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ResetPassword: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Reset password validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	err := h.resetPassword(ctx, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) resetPassword(ctx context.Context, req ResetPasswordRequestBody) error {
	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(err, http.StatusBadRequest, fmt.Sprintf("ResetPassword: Token hash not found: %s", hashedTokenStr))
		}
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to query password reset token by hash %s", hashedTokenStr))
	}

	if tokenRecord.UsedAt.Valid {
		return errlib.New(fmt.Errorf("ResetPassword: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest, "Token already used")
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		return errlib.New(fmt.Errorf("ResetPassword: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest, "Token expired")
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to hash new password")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to begin transaction")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ResetPassword: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	if err := qtx.UpdateUserPassword(ctx, &queries.UpdateUserPasswordParams{
		ID:           tokenRecord.UserID,
		PasswordHash: string(hashedNewPassword),
	}); err != nil {
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to update password for user ID %s", tokenRecord.UserID))
	}

	if err := qtx.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID); err != nil {
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to mark reset token ID %s as used", tokenRecord.ID))
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, tokenRecord.UserID); err != nil {
		errlib.LogError(errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ResetPassword: Failed to delete refresh tokens for user ID %s, but password reset was successful", tokenRecord.UserID)))
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(err, http.StatusInternalServerError, "ResetPassword: Failed to commit transaction")
	}

	return nil
}
