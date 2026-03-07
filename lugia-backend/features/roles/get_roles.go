// Feature doc: docs/features/rbac.md
package roles

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
)

var GetRolesOp = huma.Operation{
	OperationID: "get-roles",
	Method:      http.MethodGet,
	Path:        "/roles",
}

// GetUsersRolesOp serves the same handler at /users/roles for the user
// management page (requires users.view instead of roles.view).
var GetUsersRolesOp = huma.Operation{
	OperationID: "get-users-roles",
	Method:      http.MethodGet,
	Path:        "/users/roles",
}

type Permission struct {
	ID          string `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type RoleInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsDefault   bool         `json:"is_default"`
	Permissions []Permission `json:"permissions" nullable:"false"`
}

type GetRolesInput struct{}

type GetRolesResponse struct {
	Roles []RoleInfo `json:"roles" nullable:"false"`
}

type GetRolesOutput struct {
	Body GetRolesResponse
}

func (h *RolesHandler) GetRoles(ctx context.Context, input *GetRolesInput) (*GetRolesOutput, error) {
	tenantID := libctx.GetTenantID(ctx)

	response, err := h.getRoles(ctx, tenantID)
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
	return &GetRolesOutput{Body: *response}, nil
}

func (h *RolesHandler) getRoles(ctx context.Context, tenantID pgtype.UUID) (*GetRolesResponse, error) {
	rolesWithPermissions, err := h.q.GetTenantRolesWithPermissions(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			response := &GetRolesResponse{
				Roles: []RoleInfo{},
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetRoles: failed to get roles with permissions: %w", err), http.StatusInternalServerError, "")
	}

	var roleOrder []string
	roleMap := make(map[string]*RoleInfo)
	for _, row := range rolesWithPermissions {
		roleID := row.ID.String()

		if _, exists := roleMap[roleID]; !exists {
			roleOrder = append(roleOrder, roleID)
			roleMap[roleID] = &RoleInfo{
				ID:          roleID,
				Name:        row.Name,
				Description: row.Description.String,
				IsDefault:   row.IsDefault,
				Permissions: []Permission{},
			}
		}

		if row.PermissionDescription.Valid {
			permission := Permission{
				ID:          row.PermissionID.String(),
				Resource:    row.Resource.String,
				Action:      row.Action.String,
				Description: row.PermissionDescription.String,
			}
			roleMap[roleID].Permissions = append(roleMap[roleID].Permissions, permission)
		}
	}

	roleInfos := make([]RoleInfo, len(roleOrder))
	for i, roleID := range roleOrder {
		roleInfos[i] = *roleMap[roleID]
	}

	response := &GetRolesResponse{
		Roles: roleInfos,
	}

	return response, nil
}
