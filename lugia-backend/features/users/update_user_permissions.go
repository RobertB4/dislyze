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
	"lugia/queries"
)

type UpdateUserRolesRequestBody struct {
	RoleIDs []pgtype.UUID `json:"role_ids"`
}

func (r *UpdateUserRolesRequestBody) Validate() error {
	if len(r.RoleIDs) == 0 {
		return fmt.Errorf("users need at least one role")
	}
	return nil
}

// difference returns elements that are in slice1 but not in slice2
func difference(slice1, slice2 []pgtype.UUID) []pgtype.UUID {
	set := make(map[string]struct{})
	for _, item := range slice2 {
		set[item.String()] = struct{}{}
	}

	var result []pgtype.UUID
	for _, item := range slice1 {
		if _, exists := set[item.String()]; !exists {
			result = append(result, item)
		}
	}
	return result
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

	validRoleIDs, err := h.q.ValidateRolesBelongToTenant(ctx, &queries.ValidateRolesBelongToTenantParams{
		Column1:  req.RoleIDs,
		TenantID: requestingTenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to validate roles belong to tenant: %w", err), http.StatusInternalServerError, "")
	}
	if len(validRoleIDs) != len(req.RoleIDs) {
		return errlib.New(fmt.Errorf("UpdateUserPermissions: some roles don't belong to tenant"), http.StatusBadRequest, "")
	}

	currentRoleIDs, err := h.q.GetUserRoleIDs(ctx, &queries.GetUserRoleIDsParams{
		UserID:   targetUserID,
		TenantID: requestingTenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to get current user roles: %w", err), http.StatusInternalServerError, "")
	}

	toAdd := difference(req.RoleIDs, currentRoleIDs)
	toRemove := difference(currentRoleIDs, req.RoleIDs)

	if len(toRemove) > 0 || len(toAdd) > 0 {
		tx, err := h.dbConn.Begin(ctx)
		if err != nil {
			return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		}
		defer tx.Rollback(ctx)

		qtx := h.q.WithTx(tx)

		if len(toRemove) > 0 {
			err = qtx.RemoveRolesFromUser(ctx, &queries.RemoveRolesFromUserParams{
				UserID:   targetUserID,
				TenantID: requestingTenantID,
				Column3:  toRemove,
			})
			if err != nil {
				return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to remove roles: %w", err), http.StatusInternalServerError, "")
			}
		}

		if len(toAdd) > 0 {
			addRolesInput := make([]*queries.AddRolesToUserParams, len(toAdd))
			for i, roleID := range toAdd {
				addRolesInput[i] = &queries.AddRolesToUserParams{
					UserID:   targetUserID,
					RoleID:   roleID,
					TenantID: requestingTenantID,
				}
			}

			_, err = qtx.AddRolesToUser(ctx, addRolesInput)
			if err != nil {
				return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to add roles: %w", err), http.StatusInternalServerError, "")
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return errlib.New(fmt.Errorf("UpdateUserPermissions: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
		}
	}

	return nil
}
