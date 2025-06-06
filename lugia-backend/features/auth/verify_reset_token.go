package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lugia/lib/errlib"
	"lugia/lib/responder"

	"github.com/jackc/pgx/v5"
)

type VerifyResetTokenRequest struct {
	Token string `json:"token"`
}

func (r *VerifyResetTokenRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	if r.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (h *AuthHandler) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyResetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode verify reset token request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Verify reset token validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	tokenHash := sha256.Sum256([]byte(req.Token))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	tokenRecord, err := h.queries.GetPasswordResetTokenByHash(ctx, hashedTokenStr)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			internalErr := errlib.New(err, http.StatusBadRequest, fmt.Sprintf("VerifyResetToken: Token hash not found: %s", hashedTokenStr))
			errlib.LogError(internalErr)
			responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
			return
		}
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to query password reset token by hash %s", hashedTokenStr))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	if tokenRecord.UsedAt.Valid {
		internalErr := errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s already used at %v", tokenRecord.ID, tokenRecord.UsedAt.Time), http.StatusBadRequest, "Token already used")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if time.Now().After(tokenRecord.ExpiresAt.Time) {
		internalErr := errlib.New(fmt.Errorf("VerifyResetToken: Token ID %s expired at %v", tokenRecord.ID, tokenRecord.ExpiresAt.Time), http.StatusBadRequest, "Token expired")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	user, err := h.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: User ID %s for valid token %s not found", tokenRecord.UserID, tokenRecord.ID))
			errlib.LogError(internalErr)
			responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
			return
		}
		internalErr := errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("VerifyResetToken: Failed to get user email for user ID %s", tokenRecord.UserID))
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusInternalServerError, ""))
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, map[string]string{"email": user.Email})
}