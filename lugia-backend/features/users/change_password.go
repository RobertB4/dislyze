// Feature doc: docs/features/profile-management.md
package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
	"lugia/queries"
)

var ChangePasswordOp = huma.Operation{
	OperationID: "change-password",
	Method:      http.MethodPost,
	Path:        "/me/change-password",
}

type ChangePasswordInput struct {
	Body ChangePasswordRequestBody
}

type ChangePasswordRequestBody struct {
	CurrentPassword    string `json:"current_password"` // #nosec G117 -- intentional: request body, not a leaked secret
	NewPassword        string `json:"new_password"`    // #nosec G117
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

func (h *UsersHandler) ChangePassword(ctx context.Context, input *ChangePasswordInput) (*struct{}, error) {
	if err := input.Body.Validate(); err != nil {
		return nil, humautil.NewError(fmt.Errorf("change password validation failed: %w", err), http.StatusBadRequest)
	}

	userID := libctx.GetUserID(ctx)
	err := h.changePassword(ctx, userID, input.Body)
	if err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			if appErr.Message != "" {
				return nil, humautil.NewErrorWithDetail(err, appErr.StatusCode, appErr.Message)
			}
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}
	return nil, nil
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
