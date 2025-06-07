package users

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/queries"
)

func (h *UsersHandler) VerifyChangeEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	if token == "" {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: token is missing"), http.StatusBadRequest, "無効または期限切れのトークンです。")
		responder.RespondWithError(w, appErr)
		return
	}

	hash := sha256.Sum256([]byte(token))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
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
			appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: invalid or expired token: %w", err), http.StatusBadRequest, "無効または期限切れのトークンです。")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to get email change token: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if emailChangeToken.ExpiresAt.Time.Before(time.Now()) {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: token expired at %s", emailChangeToken.ExpiresAt.Time), http.StatusBadRequest, "無効または期限切れのトークンです。")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := qtx.UpdateUserEmail(ctx, &queries.UpdateUserEmailParams{
		ID:    emailChangeToken.UserID,
		Email: emailChangeToken.NewEmail,
	}); err != nil {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to update user email: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := qtx.MarkEmailChangeTokenAsUsed(ctx, emailChangeToken.ID); err != nil {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to mark token as used: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, emailChangeToken.UserID); err != nil {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to invalidate sessions: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		appErr := errlib.New(fmt.Errorf("VerifyChangeEmail: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}