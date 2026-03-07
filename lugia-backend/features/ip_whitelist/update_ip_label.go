// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
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

	_, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
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

	err = h.q.UpdateIPWhitelistLabel(ctx, &queries.UpdateIPWhitelistLabelParams{
		Label:    label,
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	return nil
}
