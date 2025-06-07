package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/queries"
	"lugia/queries_pregeneration"
)

type UpdateUserRoleRequest struct {
	Role queries_pregeneration.UserRole `json:"role"`
}

func (r *UpdateUserRoleRequest) Validate() error {
	r.Role = queries_pregeneration.UserRole(strings.TrimSpace(strings.ToLower(string(r.Role))))
	if r.Role == "" {
		return fmt.Errorf("role is required")
	}
	if r.Role != queries_pregeneration.UserRole("admin") && r.Role != queries_pregeneration.UserRole("editor") {
		return fmt.Errorf("invalid role specified, must be 'admin' or 'editor'")
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

	var req UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: failed to decode request: %w", err), http.StatusBadRequest, ""))
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: validation failed: %w", err), http.StatusBadRequest, ""))
		return
	}

	requestingUserID := libctx.GetUserID(ctx)
	requestingTenantID := libctx.GetTenantID(ctx)

	if requestingUserID == targetUserID {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: user %s attempting to update their own role", requestingUserID.String()), http.StatusBadRequest, ""))
		return
	}

	targetUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: target user with ID %s not found: %w", userIDStr, err), http.StatusNotFound, ""))
			return
		}
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: failed to get target user %s: %w", userIDStr, err), http.StatusInternalServerError, ""))
		return
	}

	if requestingTenantID != targetUser.TenantID {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: requesting user %s (tenant %s) attempting to update user %s (tenant %s) in different tenant", requestingUserID.String(), requestingTenantID.String(), targetUserID.String(), targetUser.TenantID.String()), http.StatusForbidden, ""))
		return
	}

	params := queries.UpdateUserRoleParams{
		Role:     req.Role,
		ID:       targetUserID,
		TenantID: requestingTenantID,
	}

	err = h.q.UpdateUserRole(ctx, &params)
	if err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("UpdateUserPermissions: failed to update user role: %w", err), http.StatusInternalServerError, ""))
		return
	}

	w.WriteHeader(http.StatusOK)
}