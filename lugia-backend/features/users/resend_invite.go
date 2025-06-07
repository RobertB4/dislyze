package users

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func (h *UsersHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := r.PathValue("userID")

	if !h.resendInviteRateLimiter.Allow(targetUserIDStr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for user %s resend invite", targetUserIDStr), http.StatusTooManyRequests, "招待メールの再送信は、ユーザーごとに5分間に1回のみ可能です。しばらくしてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: invalid target userID format '%s': %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	invokerDBUser, err := h.q.GetUserByID(ctx, invokerUserID)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to get invoker's user details for UserID %s: %w", invokerUserID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	if invokerDBUser == nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: invoker user not found for UserID %s", invokerUserID.String()), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("ResendInvite: target user with ID %s not found: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to get target user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if invokerTenantID != targetDBUser.TenantID {
		appErr := errlib.New(fmt.Errorf("ResendInvite: invoker %s (tenant %s) attempting to resend invite for user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()), http.StatusForbidden, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if targetDBUser.Status != "pending_verification" {
		appErr := errlib.New(fmt.Errorf("ResendInvite: target user %s status is '%s', expected 'pending_verification'", targetUserIDStr, targetDBUser.Status), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ResendInvite: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteInvitationTokensByUserIDAndTenantID(ctx, &queries.DeleteInvitationTokensByUserIDAndTenantIDParams{
		UserID:   targetUserID,
		TenantID: targetDBUser.TenantID,
	}); err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to delete existing invitation tokens for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to generate random bytes for invitation token: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	plaintextToken := base64.URLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(plaintextToken))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])
	expiresAt := time.Now().Add(48 * time.Hour) // 2 days

	_, err = qtx.CreateInvitationToken(ctx, &queries.CreateInvitationTokenParams{
		TokenHash: hashedTokenStr,
		TenantID:  targetDBUser.TenantID,
		UserID:    targetUserID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: CreateInvitationToken failed for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	subject := fmt.Sprintf("%sさんから%s様へのdislyzeへのご招待", invokerDBUser.Name, targetDBUser.Name)
	invitationLink := fmt.Sprintf("%s/auth/accept-invite?token=%s&inviter_name=%s&invited_email=%s",
		h.env.FrontendURL,
		plaintextToken,
		url.QueryEscape(invokerDBUser.Name),
		url.QueryEscape(targetDBUser.Email))

	plainTextContent := fmt.Sprintf("%s様、\n\n%sさんがあなたをdislyzeに招待しています。\n\n以下のリンクをクリックして登録を完了してください。\n%s\n\nこのメールにお心当たりがない場合は、無視してください。", targetDBUser.Name, invokerDBUser.Name, invitationLink)
	htmlContent := fmt.Sprintf(`<p>%s様</p>
	<p>%sさんがあなたをdislyzeに招待しています。</p>
	<p>以下のリンクをクリックして登録を完了してください。</p>
	<p><a href="%s">登録を完了する</a></p>
	<p>このメールにお心当たりがない場合は、無視してください。</p>`, targetDBUser.Name, invokerDBUser.Name, invitationLink)

	sgMailBody := sendgridlib.SendGridMailRequestBody{
		Personalizations: []sendgridlib.SendGridPersonalization{
			{
				To:      []sendgridlib.SendGridEmailAddress{{Email: targetDBUser.Email, Name: targetDBUser.Name}},
				Subject: subject,
			},
		},
		From:    sendgridlib.SendGridEmailAddress{Email: sendgridlib.SendGridFromEmail, Name: sendgridlib.SendGridFromName},
		Content: []sendgridlib.SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to marshal SendGrid request body for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: SendGrid API call failed for user %s: %w.", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		appErr := errlib.New(fmt.Errorf("ResendInvite: SendGrid API returned error status code %d for user %s. Body: %s.", sgResponse.StatusCode, targetUserIDStr, sgResponse.Body), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		appErr := errlib.New(fmt.Errorf("ResendInvite: failed to commit transaction for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}
