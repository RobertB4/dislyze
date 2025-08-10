package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/queries"
)

func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := r.PathValue("userID")

	if !h.deleteUserRateLimiter.Allow(targetUserIDStr, r) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for user %s delete", targetUserIDStr), http.StatusTooManyRequests, "ユーザー削除の操作は制限されています。しばらくしてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: invalid target userID format '%s': %w", targetUserIDStr, err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	err := h.deleteUser(ctx, targetUserID, invokerUserID, invokerTenantID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UsersHandler) deleteUser(ctx context.Context, targetUserID, invokerUserID, invokerTenantID pgtype.UUID) error {

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("DeleteUser: target user with ID %s not found: %w", targetUserID.String(), err), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("DeleteUser: failed to get target user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if invokerTenantID != targetDBUser.TenantID {
		return errlib.New(fmt.Errorf("DeleteUser: invoker %s (tenant %s) attempting to delete user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()), http.StatusForbidden, "")
	}

	if invokerUserID == targetUserID {
		return errlib.New(fmt.Errorf("DeleteUser: user %s attempting to delete themselves", invokerUserID.String()), http.StatusConflict, "自分自身を削除することはできません。")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("DeleteUser: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteInvitationTokensByUserIDAndTenantID(ctx, &queries.DeleteInvitationTokensByUserIDAndTenantIDParams{
		UserID:   targetUserID,
		TenantID: targetDBUser.TenantID,
	}); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to delete invitation tokens for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, targetUserID); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to delete refresh tokens for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if err := qtx.DeleteUser(ctx, targetUserID); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to delete user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to commit transaction for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	return nil
}