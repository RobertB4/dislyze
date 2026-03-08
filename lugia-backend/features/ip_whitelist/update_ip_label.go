// Feature doc: docs/features/ip-whitelisting.md, docs/features/audit-logging.md
package ip_whitelist

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

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var UpdateIPLabelOp = huma.Operation{
	OperationID: "update-ip-label",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/{id}/label/update",
}

type UpdateIPLabelInput struct {
	ID   string `path:"id"`
	Body UpdateLabelRequest
}

type UpdateLabelRequest struct {
	Label *string `json:"label" maxLength:"255"`
}

func (h *IPWhitelistHandler) UpdateIPLabel(ctx context.Context, input *UpdateIPLabelInput) (*struct{}, error) {
	var id pgtype.UUID
	if err := id.Scan(input.ID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid IP whitelist rule ID format: %w", err), http.StatusBadRequest)
	}

	err := h.updateIPLabel(ctx, id, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) updateIPLabel(ctx context.Context, id pgtype.UUID, req UpdateLabelRequest) error {
	tenantID := libctx.GetTenantID(ctx)

	rule, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return errlib.NewError(err, http.StatusNotFound)
		}
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	var label pgtype.Text
	if req.Label != nil && *req.Label != "" {
		label = pgtype.Text{String: *req.Label, Valid: true}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("UpdateIPLabel: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("UpdateIPLabel: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.UpdateIPWhitelistLabel(ctx, &queries.UpdateIPWhitelistLabelParams{
		Label:    label,
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		r := middleware.GetHTTPRequest(ctx)
		userID := libctx.GetUserID(ctx)
		actor, err := qtx.GetUserByID(ctx, userID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("UpdateIPLabel: failed to get actor for audit log: %w", err), http.StatusInternalServerError)
		}
		newLabel := ""
		if req.Label != nil {
			newLabel = *req.Label
		}
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actor.Name,
			"actor_email": actor.Email,
			"ip_address":  rule.IpAddress.String(),
			"new_label":   newLabel,
		})
		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceIPWhitelist),
			Action:       string(auditlog.ActionIPUpdated),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: id.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("UpdateIPLabel: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("UpdateIPLabel: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
