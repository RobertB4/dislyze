// Feature doc: docs/features/rbac.md
package roles

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/errlib"
)

var GetPermissionsOp = huma.Operation{
	OperationID: "get-permissions",
	Method:      http.MethodGet,
	Path:        "/roles/permissions",
}

type GetPermissionsInput struct{}

type GetPermissionsResponse struct {
	Permissions []Permission `json:"permissions" nullable:"false"`
}

type GetPermissionsOutput struct {
	Body GetPermissionsResponse
}

func (h *RolesHandler) GetPermissions(ctx context.Context, input *GetPermissionsInput) (*GetPermissionsOutput, error) {
	response, err := h.getPermissions(ctx)
	if err != nil {
		return nil, err
	}
	return &GetPermissionsOutput{Body: *response}, nil
}

func (h *RolesHandler) getPermissions(ctx context.Context) (*GetPermissionsResponse, error) {
	permissions, err := h.q.GetAllPermissions(ctx)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			response := &GetPermissionsResponse{
				Permissions: []Permission{},
			}
			return response, nil
		}
		return nil, errlib.NewError(fmt.Errorf("GetPermissions: failed to get permissions: %w", err), http.StatusInternalServerError)
	}

	permissionInfos := make([]Permission, len(permissions))
	for i, permission := range permissions {
		permissionInfos[i] = Permission{
			ID:          permission.ID.String(),
			Resource:    permission.Resource,
			Action:      permission.Action,
			Description: permission.Description,
		}
	}

	response := &GetPermissionsResponse{
		Permissions: permissionInfos,
	}

	return response, nil
}
