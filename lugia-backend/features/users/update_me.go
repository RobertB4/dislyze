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

type UpdateMeRequest struct {
	Name string `json:"name"`
}

func (r *UpdateMeRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *UsersHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)

	var req UpdateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateMe: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("UpdateMe: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateMe: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := h.q.UpdateUserName(ctx, &queries.UpdateUserNameParams{
		Name: req.Name,
		ID:   userID,
	}); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateMe: failed to update user name for user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}