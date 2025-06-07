package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/lib/sendgridlib"
	"lugia/queries"
)

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
}

func (r *ChangeEmailRequest) Validate() error {
	r.NewEmail = strings.TrimSpace(r.NewEmail)
	if r.NewEmail == "" {
		return fmt.Errorf("new email is required")
	}
	if !strings.ContainsRune(r.NewEmail, '@') {
		return fmt.Errorf("new email is invalid")
	}
	return nil
}

func (h *UsersHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)

	var req ChangeEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangeEmail: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ChangeEmail: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("ChangeEmail: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.changeEmail(ctx, userID, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UsersHandler) changeEmail(ctx context.Context, userID pgtype.UUID, req ChangeEmailRequest) error {
	existingUser, err := h.q.GetUserByEmail(ctx, req.NewEmail)
	if err == nil && existingUser != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: email %s is already in use", req.NewEmail), http.StatusConflict, "このメールアドレスは既に使用されています。")
	}
	if err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to check if email exists: %w", err), http.StatusInternalServerError, "")
	}

	if !h.changeEmailRateLimiter.Allow(userID.String()) {
		return errlib.New(fmt.Errorf("ChangeEmail: rate limit exceeded for user %s", userID.String()), http.StatusTooManyRequests, "メールアドレス変更の試行回数が上限を超えました。しばらくしてから再度お試しください。")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to generate random token: %w", err), http.StatusInternalServerError, "")
	}
	plaintextToken := base64.URLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(plaintextToken))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ChangeEmail: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteEmailChangeTokensByUserID(ctx, userID); err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to delete existing tokens: %w", err), http.StatusInternalServerError, "")
	}

	if err := qtx.CreateEmailChangeToken(ctx, &queries.CreateEmailChangeTokenParams{
		UserID:    userID,
		NewEmail:  req.NewEmail,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to create token: %w", err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	verificationLink := fmt.Sprintf("%s/verify/change-email?token=%s", h.env.FrontendURL, plaintextToken)

	plainTextContent := fmt.Sprintf("メールアドレス変更のリクエストを受け取りました。\n\n以下のリンクをクリックしてメールアドレスの変更を完了してください：\n%s\n\nこのメールにお心当たりがない場合は、無視してください。", verificationLink)
	htmlContent := fmt.Sprintf(`<p>メールアドレス変更のリクエストを受け取りました。</p>
<p>以下のリンクをクリックしてメールアドレスの変更を完了してください：</p>
<p><a href="%s">%s</a></p>
<p>このメールにお心当たりがない場合は、無視してください。</p>`, verificationLink, verificationLink)

	sgMailBody := sendgridlib.SendGridMailRequestBody{
		Personalizations: []sendgridlib.SendGridPersonalization{{
			To:      []sendgridlib.SendGridEmailAddress{{Email: req.NewEmail}},
			Subject: "メールアドレス変更の確認",
		}},
		From:    sendgridlib.SendGridEmailAddress{Email: sendgridlib.SendGridFromEmail, Name: sendgridlib.SendGridFromName},
		Content: []sendgridlib.SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: failed to marshal SendGrid request: %w", err), http.StatusInternalServerError, "")
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	response, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.New(fmt.Errorf("ChangeEmail: SendGrid API call failed: %w", err), http.StatusInternalServerError, "")
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errlib.New(fmt.Errorf("ChangeEmail: SendGrid API returned error status code %d. Body: %s", response.StatusCode, response.Body), http.StatusInternalServerError, "")
	}

	return nil
}
