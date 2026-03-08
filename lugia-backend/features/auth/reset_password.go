// Feature doc: docs/features/authentication.md, docs/features/audit-logging.md
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"dislyze/jirachi/auditlog"
	jirachiAuthz "dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var ResetPasswordOp = huma.Operation{
	OperationID: "reset-password",
	Method:      http.MethodPost,
	Path:        "/auth/reset-password",
}

type ResetPasswordInput struct {
	Body ResetPasswordRequestBody
}

type ResetPasswordRequestBody struct {
	Token           string `json:"token" minLength:"1"`
	Password        string `json:"password" minLength:"8"` // #nosec G117 -- intentional: request body, not a leaked secret
	PasswordConfirm string `json:"password_confirm" minLength:"1"`
}

func (r *ResetPasswordRequestBody) Resolve(ctx huma.Context) []error {
	if r.Password != r.PasswordConfirm {
		return []error{fmt.Errorf("passwords do not match")}
	}
	return nil
}

func (h *AuthHandler) ResetPassword(ctx context.Context, input *ResetPasswordInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)

	err := h.resetPassword(ctx, input.Body, r)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *AuthHandler) resetPassword(ctx context.Context, req ResetPasswordRequestBody, r *http.Request) error {
	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.NewError(fmt.Errorf("ResetPassword: token hash not found: %s: %w", hashedTokenStr, err), http.StatusBadRequest)
		}
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to query password reset token by hash %s: %w", hashedTokenStr, err), http.StatusInternalServerError)
	}

	if tokenRecord.UsedAt.Valid {
		return errlib.NewError(fmt.Errorf("ResetPassword: token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest)
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		return errlib.NewError(fmt.Errorf("ResetPassword: token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest)
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to hash new password: %w", err), http.StatusInternalServerError)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to begin transaction: %w", err), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to update password for user ID %s: %w", tokenRecord.UserID, err), http.StatusInternalServerError)
	}

	if err := qtx.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID); err != nil {
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to mark reset token ID %s as used: %w", tokenRecord.ID, err), http.StatusInternalServerError)
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, tokenRecord.UserID); err != nil {
		errlib.LogError(fmt.Errorf("ResetPassword: Failed to delete refresh tokens for user ID %s, but password reset was successful: %w", tokenRecord.UserID, err))
	}

	user, userErr := qtx.GetUserByID(ctx, tokenRecord.UserID)
	if userErr == nil {
		tenant, tenantErr := h.queries.GetTenantByID(ctx, user.TenantID)
		if tenantErr == nil {
			var ef jirachiAuthz.EnterpriseFeatures
			if err := json.Unmarshal(tenant.EnterpriseFeatures, &ef); err == nil && ef.AuditLog.Enabled {
				metadata, _ := json.Marshal(map[string]string{
					"actor_name":  user.Name,
					"actor_email": user.Email,
				})

				ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
				if err := qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
					TenantID:     tenant.ID,
					ActorID:      user.ID,
					ResourceType: string(auditlog.ResourceAuth),
					Action:       string(auditlog.ActionPasswordResetCompleted),
					Outcome:      string(auditlog.OutcomeSuccess),
					ResourceID:   pgtype.Text{},
					Metadata:     metadata,
					IpAddress:    &ipAddr,
					UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
				}); err != nil {
					errlib.LogError(fmt.Errorf("ResetPassword: failed to insert audit log: %w", err))
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("ResetPassword: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
