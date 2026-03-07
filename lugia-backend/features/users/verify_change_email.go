// Feature doc: docs/features/profile-management.md
package users

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/errlib"
	"lugia/queries"
)

var VerifyChangeEmailOp = huma.Operation{
	OperationID: "verify-change-email",
	Method:      http.MethodGet,
	Path:        "/me/verify-change-email",
}

type VerifyChangeEmailInput struct {
	Token string `query:"token"`
}

func (h *UsersHandler) VerifyChangeEmail(ctx context.Context, input *VerifyChangeEmailInput) (*struct{}, error) {
	if input.Token == "" {
		return nil, errlib.NewErrorWithDetail(fmt.Errorf("verify change email token is empty"), http.StatusBadRequest, "無効または期限切れのトークンです。")
	}

	err := h.verifyChangeEmail(ctx, input.Token)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) verifyChangeEmail(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("VerifyChangeEmail: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.q.WithTx(tx)

	emailChangeToken, err := qtx.GetEmailChangeTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.NewErrorWithDetail(fmt.Errorf("VerifyChangeEmail: invalid or expired token: %w", err), http.StatusBadRequest, "無効または期限切れのトークンです。")
		}
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to get email change token: %w", err), http.StatusInternalServerError)
	}

	if emailChangeToken.ExpiresAt.Time.Before(time.Now()) {
		return errlib.NewErrorWithDetail(fmt.Errorf("VerifyChangeEmail: token expired at %s", emailChangeToken.ExpiresAt.Time), http.StatusBadRequest, "無効または期限切れのトークンです。")
	}

	if err := qtx.UpdateUserEmail(ctx, &queries.UpdateUserEmailParams{
		ID:    emailChangeToken.UserID,
		Email: emailChangeToken.NewEmail,
	}); err != nil {
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to update user email: %w", err), http.StatusInternalServerError)
	}

	if err := qtx.MarkEmailChangeTokenAsUsed(ctx, emailChangeToken.ID); err != nil {
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to mark token as used: %w", err), http.StatusInternalServerError)
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, emailChangeToken.UserID); err != nil {
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to invalidate sessions: %w", err), http.StatusInternalServerError)
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("VerifyChangeEmail: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
