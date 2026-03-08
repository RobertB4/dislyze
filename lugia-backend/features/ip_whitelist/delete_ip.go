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

var DeleteIPOp = huma.Operation{
	OperationID: "delete-ip",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/{id}/delete",
}

type DeleteIPInput struct {
	ID string `path:"id"`
}

func (h *IPWhitelistHandler) DeleteIP(ctx context.Context, input *DeleteIPInput) (*struct{}, error) {
	var id pgtype.UUID
	if err := id.Scan(input.ID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid IP whitelist rule ID format: %w", err), http.StatusBadRequest)
	}

	err := h.deleteIP(ctx, id)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) deleteIP(ctx context.Context, id pgtype.UUID) error {
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

	ipConfig := libctx.GetIPWhitelistConfig(ctx)
	if ipConfig.Active {
		r := middleware.GetHTTPRequest(ctx)
		clientIP := iputils.ExtractClientIP(r)

		isCurrentIP, err := iputils.IsIPInCIDRList(clientIP, []string{rule.IpAddress.String()})
		if err != nil {
			return errlib.NewError(err, http.StatusInternalServerError)
		}

		if isCurrentIP {
			return errlib.NewErrorWithDetail(nil, http.StatusBadRequest, "現在使用中のIPアドレスは削除できません。")
		}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("DeleteIP: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("DeleteIP: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.RemoveIPFromWhitelist(ctx, &queries.RemoveIPFromWhitelistParams{
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
			return errlib.NewError(fmt.Errorf("DeleteIP: failed to get actor for audit log: %w", err), http.StatusInternalServerError)
		}
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actor.Name,
			"actor_email": actor.Email,
			"ip_address":  rule.IpAddress.String(),
		})
		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceIPWhitelist),
			Action:       string(auditlog.ActionIPRemoved),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: id.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("DeleteIP: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("DeleteIP: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
