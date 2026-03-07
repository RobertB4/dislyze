// Feature doc: docs/features/authentication.md
package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/errlib"
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
	Token string `json:"token" minLength:"1"`
}

type VerifyResetTokenResponse struct {
	Email string `json:"email"`
}

type VerifyResetTokenOutput struct {
	Body VerifyResetTokenResponse
}

func (h *AuthHandler) VerifyResetToken(ctx context.Context, input *VerifyResetTokenInput) (*VerifyResetTokenOutput, error) {
	email, err := h.verifyResetToken(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &VerifyResetTokenOutput{Body: VerifyResetTokenResponse{Email: email}}, nil
}

func (h *AuthHandler) verifyResetToken(ctx context.Context, req VerifyResetTokenRequestBody) (string, error) {
	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return "", errlib.NewError(fmt.Errorf("VerifyResetToken: token hash not found: %s: %w", hashedTokenStr, err), http.StatusBadRequest)
		}
		return "", errlib.NewError(fmt.Errorf("VerifyResetToken: failed to query password reset token by hash %s: %w", hashedTokenStr, err), http.StatusInternalServerError)
	}

	if tokenRecord.UsedAt.Valid {
		return "", errlib.NewError(fmt.Errorf("VerifyResetToken: token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest)
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		return "", errlib.NewError(fmt.Errorf("VerifyResetToken: token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest)
	}

	user, err := h.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return "", errlib.NewError(fmt.Errorf("VerifyResetToken: user ID %s for valid token %s not found: %w", tokenRecord.UserID, tokenRecord.ID, err), http.StatusInternalServerError)
		}
		return "", errlib.NewError(fmt.Errorf("VerifyResetToken: failed to get user email for user ID %s: %w", tokenRecord.UserID, err), http.StatusInternalServerError)
	}

	return user.Email, nil
}
