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
	"lugia/lib/authz"
)

// resourceToFeature maps every permission resource to its enterprise feature gate.
// Core resources use "" (always visible). Feature-gated resources use the feature constant.
// Every resource in the permissions table MUST have an entry here — a missing
// entry returns 500 to prevent silently exposing ungated permissions.
var resourceToFeature = map[string]authz.EnterpriseFeature{
	"tenant":       "",
	"users":        "",
	"roles":        "",
	"ip_whitelist": authz.FeatureIPWhitelist,
	"audit_log":    authz.FeatureAuditLog,
}

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
		return nil, err
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
		return nil, errlib.NewError(fmt.Errorf("GetRoles: failed to get roles with permissions: %w", err), http.StatusInternalServerError)
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
			feature, ok := resourceToFeature[row.Resource.String]
			if !ok {
				return nil, errlib.NewError(fmt.Errorf("GetRoles: permission resource %q not in resourceToFeature map", row.Resource.String), http.StatusInternalServerError)
			}
			if feature != "" && !authz.TenantHasFeature(ctx, feature) {
				continue
			}
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
