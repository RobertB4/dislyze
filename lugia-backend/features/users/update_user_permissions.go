package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
)

type UpdateUserRolesRequestBody struct {
	RoleIDs []string `json:"role_ids"`
}

func (r *UpdateUserRolesRequestBody) Validate() error {
	if len(r.RoleIDs) == 0 {
		return fmt.Errorf("ユーザーには最低1つの権限が必要です")
	}
	return nil
}

func (h *UsersHandler) UpdateUserPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("userID")
	if userIDStr == "" {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("user ID is required"), http.StatusBadRequest, ""))
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(userIDStr); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: invalid target userID format '%s': %w", userIDStr, err), http.StatusBadRequest, ""))
		return
	}

	var req UpdateUserRolesRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: failed to decode request: %w", err), http.StatusBadRequest, ""))
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: validation failed: %w", err), http.StatusBadRequest, ""))
		return
	}

	err := h.updateUserPermissions(ctx, targetUserID, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UsersHandler) updateUserPermissions(ctx context.Context, targetUserID pgtype.UUID, req UpdateUserRolesRequestBody) error {
	requestingUserID := libctx.GetUserID(ctx)
	requestingTenantID := libctx.GetTenantID(ctx)

	if requestingUserID == targetUserID {
		return errlib.New(fmt.Errorf("UpdateUserPermissions: user %s attempting to update their own role", requestingUserID.String()), http.StatusBadRequest, "")
	}

	targetUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("UpdateUserPermissions: target user with ID %s not found: %w", targetUserID.String(), err), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to get target user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if requestingTenantID != targetUser.TenantID {
		return errlib.New(fmt.Errorf("UpdateUserPermissions: requesting user %s (tenant %s) attempting to update user %s (tenant %s) in different tenant", requestingUserID.String(), requestingTenantID.String(), targetUserID.String(), targetUser.TenantID.String()), http.StatusForbidden, "")
	}

	// TODO: Implement role assignment using new role-based permission system
	// This needs to:
	// 1. Validate role IDs belong to tenant
	// 2. Get current role IDs for user
	// 3. Calculate differences (toAdd, toRemove)
	// 4. Remove roles in transaction
	// 5. Add roles in transaction
	// For now, return success to allow compilation
	_ = req.RoleIDs // Acknowledge the parameter to avoid unused variable error

	return nil
}
