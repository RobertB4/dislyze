// Feature doc: docs/features/user-management.md
package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
	"lugia/lib/middleware"
)

var DeleteUserOp = huma.Operation{
	OperationID: "delete-user",
	Method:      http.MethodPost,
	Path:        "/users/{userID}/delete",
}

type DeleteUserInput struct {
	UserID string `path:"userID"`
}

func (h *UsersHandler) DeleteUser(ctx context.Context, input *DeleteUserInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)

	if !h.deleteUserRateLimiter.Allow(input.UserID, r) {
		return nil, humautil.NewErrorWithDetail(fmt.Errorf("rate limit exceeded for delete user"), http.StatusTooManyRequests, "ユーザー削除の操作は制限されています。しばらくしてから再度お試しください。")
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(input.UserID); err != nil {
		return nil, humautil.NewError(fmt.Errorf("invalid user ID format for delete user: %w", err), http.StatusBadRequest)
	}

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	err := h.deleteUser(ctx, targetUserID, invokerUserID, invokerTenantID)
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

	if err := qtx.MarkUserDeletedAndAnonymize(ctx, targetUserID); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to anonymize user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("DeleteUser: failed to commit transaction for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	return nil
}
