package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
)

type RoleInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsDefault   bool     `json:"is_default"`
	Permissions []string `json:"permissions"`
}

type GetTenantRolesResponse struct {
	Roles []RoleInfo `json:"roles"`
}

func (h *UsersHandler) GetTenantRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawTenantID := libctx.GetTenantID(ctx)

	response, err := h.getTenantRoles(ctx, rawTenantID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *UsersHandler) getTenantRoles(ctx context.Context, tenantID pgtype.UUID) (*GetTenantRolesResponse, error) {
	rolesWithPermissions, err := h.q.GetTenantRolesWithPermissions(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			response := &GetTenantRolesResponse{
				Roles: []RoleInfo{},
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetTenantRoles: failed to get roles with permissions: %w", err), http.StatusInternalServerError, "")
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
				Permissions: []string{},
			}
		}

		if row.PermissionDescription.Valid {
			permission := row.PermissionDescription.String
			roleMap[roleID].Permissions = append(roleMap[roleID].Permissions, permission)
		}
	}

	roleInfos := make([]RoleInfo, len(roleOrder))
	for i, roleID := range roleOrder {
		roleInfos[i] = *roleMap[roleID]
	}

	response := &GetTenantRolesResponse{
		Roles: roleInfos,
	}

	return response, nil
}
