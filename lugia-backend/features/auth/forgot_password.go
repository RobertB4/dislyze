package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/lib/sendgridlib"
	"lugia/lib/utils"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"
)

type ForgotPasswordRequestBody struct {
	Email string `json:"email"`
}

func (r *ForgotPasswordRequestBody) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(r.Email, "@") {
		return fmt.Errorf("invalid email address format")
	}
	return nil
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.rateLimiter.Allow(r.RemoteAddr) {
		internalErr := errlib.New(fmt.Errorf("rate limit exceeded for forgot password: %s", r.RemoteAddr), http.StatusTooManyRequests, "Rate limit for forgot password")
		errlib.LogError(internalErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ForgotPasswordRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Failed to decode forgot password request body")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ForgotPassword: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		internalErr := errlib.New(err, http.StatusBadRequest, "Forgot password validation failed")
		errlib.LogError(internalErr)
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	err := h.forgotPassword(ctx, req)
	if err != nil {
		errlib.LogError(err)
		// Always return 200 OK for security reasons (prevent email enumeration)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) forgotPassword(ctx context.Context, req ForgotPasswordRequestBody) error {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			log.Printf("ForgotPassword: No user found for email %s", req.Email)
		} else {
			return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to get user by email %s", req.Email))
		}
		return nil
	}

	resetTokenUUID, err := utils.NewUUID()
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, "ForgotPassword: Failed to generate reset token UUID")
	}
	resetToken := resetTokenUUID.String()

	tokenHash := sha256.Sum256([]byte(resetToken))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, txErr := h.dbConn.Begin(ctx)
	if txErr != nil {
		return errlib.New(txErr, http.StatusInternalServerError, "ForgotPassword: Failed to begin transaction")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ForgotPassword: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	if err := qtx.DeletePasswordResetTokenByUserID(ctx, user.ID); err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to delete existing password reset token for user %s", user.ID))
	}

	_, createErr := qtx.CreatePasswordResetToken(ctx, &queries.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if createErr != nil {
		return errlib.New(createErr, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: Failed to create password reset token for user %s", user.ID))
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return errlib.New(commitErr, http.StatusInternalServerError, "ForgotPassword: Failed to commit transaction")
	}

	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", h.env.FrontendURL, resetToken)

	subject := "パスワードリセットのご案内 - dislyze"
	plainTextContent := fmt.Sprintf("%s様\n\ndislyzeアカウントのパスワードリセットリクエストを受け付けました。\n\n以下のリンクをクリックして、パスワードを再設定してください。このリンクは30分間有効です。\n%s\n\nこのメールにお心当たりがない場合は、無視してください。",
		user.Name, resetLink)
	htmlContent := fmt.Sprintf("<p>%s様</p>\n<p>dislyzeアカウントのパスワードリセットリクエストを受け付けました。</p>\n<p>以下のリンクをクリックして、パスワードを再設定してください。このリンクは30分間有効です。</p>\n<p><a href=\"%s\">パスワードを再設定する</a></p>\n<p>このメールにお心当たりがない場合は、無視してください。</p>",
		user.Name, resetLink)

	sgMailBody := sendgridlib.SendGridMailRequestBody{
		Personalizations: []sendgridlib.SendGridPersonalization{
			{
				To:      []sendgridlib.SendGridEmailAddress{{Email: req.Email, Name: user.Name}},
				Subject: subject,
			},
		},
		From:    sendgridlib.SendGridEmailAddress{Email: sendgridlib.SendGridFromEmail, Name: sendgridlib.SendGridFromName},
		Content: []sendgridlib.SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: failed to marshal SendGrid request body for %s", req.Email))
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: SendGrid API call failed for %s", req.Email))
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		return errlib.New(fmt.Errorf("SendGrid API returned error status code: %d, Body: %s", sgResponse.StatusCode, sgResponse.Body), http.StatusInternalServerError, fmt.Sprintf("ForgotPassword: SendGrid API error for %s", req.Email))
	}

	log.Printf("Password reset email successfully sent via SendGrid to user with id: %s", user.ID)

	return nil
}
