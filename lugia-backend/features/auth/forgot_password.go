// Feature doc: docs/features/authentication.md
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/sendgridlib"
	"dislyze/jirachi/utils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var ForgotPasswordOp = huma.Operation{
	OperationID: "forgot-password",
	Method:      http.MethodPost,
	Path:        "/auth/forgot-password",
}

type ForgotPasswordInput struct {
	Body ForgotPasswordRequestBody
}

type ForgotPasswordRequestBody struct {
	Email string `json:"email" minLength:"1" pattern:"@"`
}

func (h *AuthHandler) ForgotPassword(ctx context.Context, input *ForgotPasswordInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)

	if !h.rateLimiter.Allow(r.RemoteAddr, r) {
		errlib.LogError(fmt.Errorf("rate limit exceeded for forgot password: %s", r.RemoteAddr))
		// Always return success for security (prevent email enumeration)
		return nil, nil
	}

	if err := h.forgotPassword(ctx, input.Body); err != nil {
		errlib.LogError(err)
		// Always return success for security (prevent email enumeration)
	}

	return nil, nil
}

func (h *AuthHandler) forgotPassword(ctx context.Context, req ForgotPasswordRequestBody) error {
	user, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			log.Printf("ForgotPassword: No user found for email %s", req.Email)
		} else {
			return errlib.NewError(fmt.Errorf("ForgotPassword: failed to get user by email %s: %w", req.Email, err), http.StatusInternalServerError)
		}
		return nil
	}

	resetTokenUUID, err := utils.NewUUID()
	if err != nil {
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to generate reset token UUID: %w", err), http.StatusInternalServerError)
	}
	resetToken := resetTokenUUID.String()

	tokenHash := sha256.Sum256([]byte(resetToken))
	hashedTokenStr := fmt.Sprintf("%x", tokenHash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, txErr := h.dbConn.Begin(ctx)
	if txErr != nil {
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to begin transaction: %w", txErr), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ForgotPassword: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.queries.WithTx(tx)

	if err := qtx.DeletePasswordResetTokenByUserID(ctx, user.ID); err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to delete existing password reset token for user %s: %w", user.ID, err), http.StatusInternalServerError)
	}

	_, createErr := qtx.CreatePasswordResetToken(ctx, &queries.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if createErr != nil {
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to create password reset token for user %s: %w", user.ID, createErr), http.StatusInternalServerError)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to commit transaction: %w", commitErr), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("ForgotPassword: failed to marshal SendGrid request body for %s: %w", req.Email, err), http.StatusInternalServerError)
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ForgotPassword: SendGrid API call failed for %s: %w", req.Email, err), http.StatusInternalServerError)
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		return errlib.NewError(fmt.Errorf("ForgotPassword: SendGrid API error for %s: status code %d, body: %s", req.Email, sgResponse.StatusCode, sgResponse.Body), http.StatusInternalServerError)
	}

	log.Printf("Password reset email successfully sent via SendGrid to user with id: %s", user.ID) // #nosec G706 -- user.ID is a database UUID, not user input

	return nil
}
