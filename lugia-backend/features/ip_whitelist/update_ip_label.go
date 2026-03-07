// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
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
	Label *string `json:"label"`
}

func (r *UpdateLabelRequest) Validate() error {
	if r.Label != nil {
		trimmed := strings.TrimSpace(*r.Label)
		r.Label = &trimmed

		if len(*r.Label) > 255 {
			return errlib.New(nil, http.StatusBadRequest, "")
		}
	}

	return nil
}

func (h *IPWhitelistHandler) UpdateIPLabel(ctx context.Context, input *UpdateIPLabelInput) (*struct{}, error) {
	var id pgtype.UUID
	if err := id.Scan(input.ID); err != nil {
		return nil, humautil.NewError(fmt.Errorf("invalid IP whitelist rule ID format: %w", err), http.StatusBadRequest)
	}

	if err := input.Body.Validate(); err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusBadRequest)
	}

	err := h.updateIPLabel(ctx, id, input.Body)
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

func (h *IPWhitelistHandler) updateIPLabel(ctx context.Context, id pgtype.UUID, req UpdateLabelRequest) error {
	tenantID := libctx.GetTenantID(ctx)

	_, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return errlib.New(err, http.StatusNotFound, "")
		}
		return errlib.New(err, http.StatusInternalServerError, "")
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
		return errlib.New(err, http.StatusInternalServerError, "")
	}

	return nil
}
