package roles

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
)

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
	Permissions []Permission `json:"permissions"`
}

type GetRolesResponse struct {
	Roles []RoleInfo `json:"roles"`
}

func (h *RolesHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawTenantID := libctx.GetTenantID(ctx)

	response, err := h.getRoles(ctx, rawTenantID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
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
