package ip_whitelist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	jirachiAuthz "dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/lib/jwt"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
)

func (h *IPWhitelistHandler) EmergencyDeactivate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	if token == "" {
		appErr := errlib.New(fmt.Errorf("EmergencyDeactivate: token query parameter is required"), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.emergencyDeactivate(ctx, token)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *IPWhitelistHandler) emergencyDeactivate(ctx context.Context, token string) error {
	claims, err := jwt.ValidateEmergencyToken(token, []byte(h.env.IPWhitelistEmergencyJWTSecret))
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: invalid emergency token: %w", err), http.StatusUnauthorized, "")
	}

	currentUserID := libctx.GetUserID(ctx)
	if currentUserID != claims.UserID {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: token user mismatch"), http.StatusForbidden, "")
	}

	tokenRecord, err := h.q.GetIPWhitelistEmergencyTokenByJTI(ctx, claims.JTI)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("EmergencyDeactivate: emergency token not found"), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to get emergency token: %w", err), http.StatusInternalServerError, "")
	}

	if tokenRecord.UsedAt.Valid {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: emergency token already used"), http.StatusConflict, "")
	}

	tenant, err := h.q.GetTenantByID(ctx, claims.TenantID)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to get tenant: %w", err), http.StatusInternalServerError, "")
	}

	if len(tenant.EnterpriseFeatures) == 0 {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: tenant has no enterprise features configured"), http.StatusInternalServerError, "")
	}

	var currentFeatures jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &currentFeatures); err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to parse enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	currentFeatures.IPWhitelist.Active = false

	updatedFeaturesJSON, err := json.Marshal(currentFeatures)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to marshal enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("EmergencyDeactivate: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.UpdateTenantEnterpriseFeatures(ctx, &queries.UpdateTenantEnterpriseFeaturesParams{
		EnterpriseFeatures: updatedFeaturesJSON,
		ID:                 claims.TenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to update tenant enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.MarkIPWhitelistEmergencyTokenAsUsed(ctx, claims.JTI)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to mark emergency token as used: %w", err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
