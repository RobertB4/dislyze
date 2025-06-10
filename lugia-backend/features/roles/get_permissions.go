package roles

import (
	"context"
	"fmt"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type PermissionInfo struct {
	ID          string `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type GetPermissionsResponse struct {
	Permissions []PermissionInfo `json:"permissions"`
}

func (h *RolesHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response, err := h.getPermissions(ctx)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *RolesHandler) getPermissions(ctx context.Context) (*GetPermissionsResponse, error) {
	permissions, err := h.q.GetAllPermissions(ctx)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			response := &GetPermissionsResponse{
				Permissions: []PermissionInfo{},
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetPermissions: failed to get permissions: %w", err), http.StatusInternalServerError, "")
	}

	permissionInfos := make([]PermissionInfo, len(permissions))
	for i, permission := range permissions {
		permissionInfos[i] = PermissionInfo{
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
