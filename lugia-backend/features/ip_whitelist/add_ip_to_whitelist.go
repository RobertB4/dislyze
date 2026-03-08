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

var AddIPOp = huma.Operation{
	OperationID: "add-ip-to-whitelist",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/create",
}

type AddIPInput struct {
	Body AddIPToWhitelistRequest
}

type AddIPToWhitelistRequest struct {
	IPAddress string  `json:"ip_address" minLength:"1"`
	Label     *string `json:"label" maxLength:"255"`
}

func (r *AddIPToWhitelistRequest) Resolve(ctx huma.Context) []error {
	if _, err := iputils.ValidateCIDR(r.IPAddress); err != nil {
		return []error{fmt.Errorf("invalid IP address or CIDR: %w", err)}
	}
	return nil
}

func (h *IPWhitelistHandler) AddIPToWhitelist(ctx context.Context, input *AddIPInput) (*struct{}, error) {
	err := h.addIPToWhitelist(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) addIPToWhitelist(ctx context.Context, req AddIPToWhitelistRequest) error {
	tenantID := libctx.GetTenantID(ctx)
	userID := libctx.GetUserID(ctx)

	normalizedCIDR, err := iputils.ValidateCIDR(req.IPAddress)
	if err != nil {
		return errlib.NewError(err, http.StatusBadRequest)
	}

	prefix, err := netip.ParsePrefix(normalizedCIDR)
	if err != nil {
		return errlib.NewError(err, http.StatusBadRequest)
	}

	exists, err := h.q.CheckIPExists(ctx, &queries.CheckIPExistsParams{
		TenantID:  tenantID,
		IpAddress: prefix,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}
	if exists {
		return errlib.NewError(fmt.Errorf("AddIPToWhitelist: IP %s already exists for tenant", prefix), http.StatusBadRequest)
	}

	var label pgtype.Text
	if req.Label != nil {
		label = pgtype.Text{String: *req.Label, Valid: true}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("AddIPToWhitelist: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("AddIPToWhitelist: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	newRule, err := qtx.AddIPToWhitelist(ctx, &queries.AddIPToWhitelistParams{
		TenantID:  tenantID,
		IpAddress: prefix,
		Label:     label,
		CreatedBy: userID,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		r := middleware.GetHTTPRequest(ctx)
		actor, err := qtx.GetUserByID(ctx, userID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("AddIPToWhitelist: failed to get actor for audit log: %w", err), http.StatusInternalServerError)
		}
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actor.Name,
			"actor_email": actor.Email,
			"ip_address":  normalizedCIDR,
		})
		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceIPWhitelist),
			Action:       string(auditlog.ActionIPAdded),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: newRule.ID.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("AddIPToWhitelist: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("AddIPToWhitelist: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
