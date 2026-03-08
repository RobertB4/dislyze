// Feature doc: docs/features/profile-management.md, docs/features/audit-logging.md
package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
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
	CurrentPassword    string `json:"current_password" minLength:"1"` // #nosec G117 -- intentional: request body, not a leaked secret
	NewPassword        string `json:"new_password" minLength:"8"`    // #nosec G117
	NewPasswordConfirm string `json:"new_password_confirm" minLength:"1"`
}

func (r *ChangePasswordRequestBody) Resolve(ctx huma.Context) []error {
	if r.NewPassword != r.NewPasswordConfirm {
		return []error{fmt.Errorf("new passwords do not match")}
	}
	if r.CurrentPassword == r.NewPassword {
		return []error{fmt.Errorf("new password must be different from current password")}
	}
	return nil
}

func (h *UsersHandler) ChangePassword(ctx context.Context, input *ChangePasswordInput) (*struct{}, error) {
	userID := libctx.GetUserID(ctx)
	err := h.changePassword(ctx, userID, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) changePassword(ctx context.Context, userID pgtype.UUID, req ChangePasswordRequestBody) error {

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.NewError(fmt.Errorf("ChangePassword: user not found %s: %w", userID.String(), err), http.StatusNotFound)
		}
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return errlib.NewErrorWithDetail(fmt.Errorf("ChangePassword: current password verification failed for user %s: %w", userID.String(), err), http.StatusBadRequest, "現在のパスワードが正しくありません。")
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to hash new password for user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to begin transaction: %w", err), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to update password for user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, userID); err != nil {
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to invalidate refresh tokens for user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		tenantID := libctx.GetTenantID(ctx)
		r := middleware.GetHTTPRequest(ctx)
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  user.Name,
			"actor_email": user.Email,
		})

		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceAuth),
			Action:       string(auditlog.ActionPasswordChanged),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: userID.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangePassword: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("ChangePassword: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
