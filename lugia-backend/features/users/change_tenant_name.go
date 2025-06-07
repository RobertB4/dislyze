package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/queries"
)

type ChangeTenantNameRequest struct {
	Name string `json:"name"`
}

func (r *ChangeTenantNameRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *UsersHandler) ChangeTenantName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := libctx.GetTenantID(ctx)

	var req ChangeTenantNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangeTenantName: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ChangeTenantName: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangeTenantName: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := h.q.UpdateTenantName(ctx, &queries.UpdateTenantNameParams{
		Name: req.Name,
		ID:   tenantID,
	}); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangeTenantName: failed to update tenant name for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}
