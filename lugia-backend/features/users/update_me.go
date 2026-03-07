// Feature doc: docs/features/profile-management.md
package users

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

var UpdateMeOp = huma.Operation{
	OperationID: "update-me",
	Method:      http.MethodPost,
	Path:        "/me/change-name",
}

type UpdateMeInput struct {
	Body UpdateMeRequestBody
}

type UpdateMeRequestBody struct {
	Name string `json:"name"`
}

func (r *UpdateMeRequestBody) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *UsersHandler) UpdateMe(ctx context.Context, input *UpdateMeInput) (*struct{}, error) {
	if err := input.Body.Validate(); err != nil {
		return nil, errlib.NewError(fmt.Errorf("update me validation failed: %w", err), http.StatusBadRequest)
	}

	err := h.updateMe(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) updateMe(ctx context.Context, req UpdateMeRequestBody) error {
	userID := libctx.GetUserID(ctx)

	if err := h.q.UpdateUserName(ctx, &queries.UpdateUserNameParams{
		Name: req.Name,
		ID:   userID,
	}); err != nil {
		return errlib.NewError(fmt.Errorf("UpdateMe: failed to update user name for user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	return nil
}
