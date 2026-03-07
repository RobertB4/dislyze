// Feature doc: docs/features/profile-management.md
package users

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

var ChangeTenantNameOp = huma.Operation{
	OperationID: "change-tenant-name",
	Method:      http.MethodPost,
	Path:        "/tenant/change-name",
}

type ChangeTenantNameInput struct {
	Body ChangeTenantNameRequestBody
}

type ChangeTenantNameRequestBody struct {
	Name string `json:"name"`
}

func (r *ChangeTenantNameRequestBody) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *UsersHandler) ChangeTenantName(ctx context.Context, input *ChangeTenantNameInput) (*struct{}, error) {
	if err := input.Body.Validate(); err != nil {
		return nil, errlib.NewError(fmt.Errorf("change tenant name validation failed: %w", err), http.StatusBadRequest)
	}

	tenantID := libctx.GetTenantID(ctx)
	err := h.changeTenantName(ctx, tenantID, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) changeTenantName(ctx context.Context, tenantID pgtype.UUID, req ChangeTenantNameRequestBody) error {
	if err := h.q.UpdateTenantName(ctx, &queries.UpdateTenantNameParams{
		Name: req.Name,
		ID:   tenantID,
	}); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to update tenant name for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError)
	}

	return nil
}
