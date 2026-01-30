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
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"

	"dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"dislyze/jirachi/sendgridlib"
	"lugia/queries"
)

func (h *UsersHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := r.PathValue("userID")

	if !h.resendInviteRateLimiter.Allow(targetUserIDStr, r) {
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

	err := h.resendInvite(ctx, targetUserID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UsersHandler) resendInvite(ctx context.Context, targetUserID pgtype.UUID) error {
	invokerTenantID := libctx.GetTenantID(ctx)

	tenant, err := h.q.GetTenantByID(ctx, invokerTenantID)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to get tenant: %w", err), http.StatusInternalServerError, "")
	}

	if tenant.AuthMethod == "sso" {
		return h.resendSSOInvite(ctx, targetUserID, tenant)
	}

	return h.resendPasswordInvite(ctx, targetUserID)
}

func (h *UsersHandler) resendPasswordInvite(ctx context.Context, targetUserID pgtype.UUID) error {
	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	invokerDBUser, err := h.q.GetUserByID(ctx, invokerUserID)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to get invoker's user details for UserID %s: %w", invokerUserID.String(), err), http.StatusInternalServerError, "")
	}
	if invokerDBUser == nil {
		return errlib.New(fmt.Errorf("ResendInvite: invoker user not found for UserID %s", invokerUserID.String()), http.StatusInternalServerError, "")
	}

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("ResendInvite: target user with ID %s not found: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
		}
		return errlib.New(fmt.Errorf("ResendInvite: failed to get target user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if invokerTenantID != targetDBUser.TenantID {
		return errlib.New(fmt.Errorf("ResendInvite: invoker %s (tenant %s) attempting to resend invite for user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()), http.StatusForbidden, "")
	}

	if targetDBUser.Status != "pending_verification" {
		return errlib.New(fmt.Errorf("ResendInvite: target user %s status is '%s', expected 'pending_verification'", targetUserID.String(), targetDBUser.Status), http.StatusInternalServerError, "")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
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
		return errlib.New(fmt.Errorf("ResendInvite: failed to delete existing invitation tokens for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to generate random bytes for invitation token: %w", err), http.StatusInternalServerError, "")
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
		return errlib.New(fmt.Errorf("ResendInvite: CreateInvitationToken failed for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
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
		return errlib.New(fmt.Errorf("ResendInvite: failed to marshal SendGrid request body for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: SendGrid API call failed for user %s: %w.", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		return errlib.New(fmt.Errorf("ResendInvite: SendGrid API returned error status code %d for user %s. Body: %s.", sgResponse.StatusCode, targetUserID.String(), sgResponse.Body), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to commit transaction for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	return nil
}

func (h *UsersHandler) resendSSOInvite(ctx context.Context, targetUserID pgtype.UUID, tenant *queries.Tenant) error {
	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	var enterpriseFeatures authz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to parse enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	if !enterpriseFeatures.SSO.Enabled {
		return errlib.New(fmt.Errorf("ResendInvite: SSO not enabled for tenant"), http.StatusBadRequest, "")
	}

	invokerDBUser, err := h.q.GetUserByID(ctx, invokerUserID)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: failed to get invoker's user details for UserID %s: %w", invokerUserID.String(), err), http.StatusInternalServerError, "")
	}
	if invokerDBUser == nil {
		return errlib.New(fmt.Errorf("ResendInvite: invoker user not found for UserID %s", invokerUserID.String()), http.StatusInternalServerError, "")
	}

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("ResendInvite: target user with ID %s not found: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
		}
		return errlib.New(fmt.Errorf("ResendInvite: failed to get target user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if invokerTenantID != targetDBUser.TenantID {
		return errlib.New(fmt.Errorf("ResendInvite: invoker %s (tenant %s) attempting to resend invite for user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()), http.StatusForbidden, "")
	}

	if targetDBUser.Status != "pending_verification" {
		return errlib.New(fmt.Errorf("ResendInvite: target user %s status is '%s', expected 'pending_verification'", targetUserID.String(), targetDBUser.Status), http.StatusInternalServerError, "")
	}

	subject := fmt.Sprintf("%sさんから%s様へのdislyzeへのご招待", invokerDBUser.Name, targetDBUser.Name)
	invitationLink := fmt.Sprintf("%s/auth/sso/login?email=%s",
		h.env.FrontendURL,
		url.QueryEscape(targetDBUser.Email))

	plainTextContent := fmt.Sprintf("%s様、\n\n%sさんがあなたをdislyzeに招待しています。\n\n以下のリンクをクリックしてSSOでログインしてください。\n%s\n\nこのメールにお心当たりがない場合は、無視してください。", targetDBUser.Name, invokerDBUser.Name, invitationLink)
	htmlContent := fmt.Sprintf(`<p>%s様</p>
	<p>%sさんがあなたをdislyzeに招待しています。</p>
	<p>以下のリンクをクリックしてSSOでログインしてください。</p>
	<p><a href="%s">SSOでログインする</a></p>
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
		return errlib.New(fmt.Errorf("ResendInvite: failed to marshal SendGrid request body for user %s: %w", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.New(fmt.Errorf("ResendInvite: SendGrid API call failed for user %s: %w.", targetUserID.String(), err), http.StatusInternalServerError, "")
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		return errlib.New(fmt.Errorf("ResendInvite: SendGrid API returned error status code %d for user %s. Body: %s.", sgResponse.StatusCode, targetUserID.String(), sgResponse.Body), http.StatusInternalServerError, "")
	}

	return nil
}
