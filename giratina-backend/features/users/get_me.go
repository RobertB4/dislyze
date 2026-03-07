// Feature doc: docs/features/authentication.md
package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"

	"giratina/lib/humautil"
)

var GetMeOp = huma.Operation{
	OperationID: "get-me",
	Method:      http.MethodGet,
	Path:        "/me",
}

type MeResponse struct {
	TenantName string `json:"tenant_name"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	UserName   string `json:"user_name"`
}

type GetMeInput struct{}

type GetMeOutput struct {
	Body MeResponse
}

func (h *UsersHandler) GetMe(ctx context.Context, input *GetMeInput) (*GetMeOutput, error) {
	response, err := h.getMe(ctx)
	if err != nil {
		return nil, err
	}

	return &GetMeOutput{Body: *response}, nil
}

func (h *UsersHandler) getMe(ctx context.Context) (*MeResponse, error) {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	user, err := h.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, humautil.NewError(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusUnauthorized)
		}
		return nil, humautil.NewError(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	tenant, err := h.queries.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, humautil.NewError(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusUnauthorized)
		}
		return nil, humautil.NewError(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError)
	}

	response := &MeResponse{
		TenantName: tenant.Name,
		UserID:     user.ID.String(),
		Email:      user.Email,
		UserName:   user.Name,
	}

	return response, nil
}
