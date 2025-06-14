package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/queries"
)

type ChangePasswordRequestBody struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	NewPasswordConfirm string `json:"new_password_confirm"`
}

func (r *ChangePasswordRequestBody) Validate() error {
	r.CurrentPassword = strings.TrimSpace(r.CurrentPassword)
	r.NewPassword = strings.TrimSpace(r.NewPassword)
	r.NewPasswordConfirm = strings.TrimSpace(r.NewPasswordConfirm)

	if r.CurrentPassword == "" {
		return fmt.Errorf("current password is required")
	}
	if r.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	if len(r.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters long")
	}
	if r.NewPassword != r.NewPasswordConfirm {
		return fmt.Errorf("new passwords do not match")
	}
	if r.CurrentPassword == r.NewPassword {
		return fmt.Errorf("new password must be different from current password")
	}
	return nil
}

func (h *UsersHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)

	var req ChangePasswordRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangePassword: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ChangePassword: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangePassword: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.changePassword(ctx, userID, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UsersHandler) changePassword(ctx context.Context, userID pgtype.UUID, req ChangePasswordRequestBody) error {

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("ChangePassword: user not found %s: %w", userID.String(), err), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("ChangePassword: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: current password verification failed for user %s: %w", userID.String(), err), http.StatusBadRequest, "現在のパスワードが正しくありません。")
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: failed to hash new password for user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ChangePassword: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.UpdateUserPassword(ctx, &queries.UpdateUserPasswordParams{
		PasswordHash: string(newPasswordHash),
		ID:           userID,
	}); err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: failed to update password for user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, userID); err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: failed to invalidate refresh tokens for user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("ChangePassword: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
