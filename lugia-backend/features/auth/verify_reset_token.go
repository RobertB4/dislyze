// Feature doc: docs/features/authentication.md
package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
)

var VerifyResetTokenOp = huma.Operation{
	OperationID: "verify-reset-token",
	Method:      http.MethodPost,
	Path:        "/auth/verify-reset-token",
}

type VerifyResetTokenInput struct {
	Body VerifyResetTokenRequestBody
}

type VerifyResetTokenRequestBody struct {
	Token string `json:"token"`
}

type VerifyResetTokenResponse struct {
	Email string `json:"email"`
}

type VerifyResetTokenOutput struct {
	Body VerifyResetTokenResponse
}

func (r *VerifyResetTokenRequestBody) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	if r.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (h *AuthHandler) VerifyResetToken(ctx context.Context, input *VerifyResetTokenInput) (*VerifyResetTokenOutput, error) {
	if err := input.Body.Validate(); err != nil {
		return nil, humautil.NewError(fmt.Errorf("verify reset token validation failed: %w", err), http.StatusBadRequest)
	}

	email, err := h.verifyResetToken(ctx, input.Body)
	if err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			if appErr.Message != "" {
				return nil, humautil.NewErrorWithDetail(err, appErr.StatusCode, appErr.Message)
			}
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}

	return &VerifyResetTokenOutput{Body: VerifyResetTokenResponse{Email: email}}, nil
}

func (h *AuthHandler) verifyResetToken(ctx context.Context, req VerifyResetTokenRequestBody) (string, error) {
	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return "", errlib.New(err, http.StatusBadRequest, fmt.Sprintf("VerifyResetToken: Token hash not found: %s", hashedTokenStr))
		}
		return "", errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to query password reset token by hash %s", hashedTokenStr))
	}

	if tokenRecord.UsedAt.Valid {
		return "", errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest, "Token already used")
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		return "", errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest, "Token expired")
	}

	user, err := h.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return "", errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: User ID %s for valid token %s not found", tokenRecord.UserID, tokenRecord.ID))
		}
		return "", errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to get user email for user ID %s", tokenRecord.UserID))
	}

	return user.Email, nil
}
