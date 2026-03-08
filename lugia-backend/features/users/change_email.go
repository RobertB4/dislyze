// Feature doc: docs/features/profile-management.md, docs/features/audit-logging.md
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
	"net/netip"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/sendgridlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var ChangeEmailOp = huma.Operation{
	OperationID: "change-email",
	Method:      http.MethodPost,
	Path:        "/me/change-email",
}

type ChangeEmailInput struct {
	Body ChangeEmailRequestBody
}

type ChangeEmailRequestBody struct {
	NewEmail string `json:"new_email" minLength:"1" pattern:"@"`
}

func (h *UsersHandler) ChangeEmail(ctx context.Context, input *ChangeEmailInput) (*struct{}, error) {
	userID := libctx.GetUserID(ctx)
	r := middleware.GetHTTPRequest(ctx)

	err := h.changeEmail(ctx, userID, input.Body, r)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) changeEmail(ctx context.Context, userID pgtype.UUID, req ChangeEmailRequestBody, r *http.Request) error {
	existingUser, err := h.q.GetUserByEmail(ctx, req.NewEmail)
	if err == nil && existingUser != nil {
		return errlib.NewErrorWithDetail(fmt.Errorf("ChangeEmail: email %s is already in use", req.NewEmail), http.StatusConflict, "このメールアドレスは既に使用されています。")
	}
	if err != nil && !errlib.Is(err, pgx.ErrNoRows) {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to check if email exists: %w", err), http.StatusInternalServerError)
	}

	if !h.changeEmailRateLimiter.Allow(userID.String(), r) {
		return errlib.NewErrorWithDetail(fmt.Errorf("ChangeEmail: rate limit exceeded for user %s", userID.String()), http.StatusTooManyRequests, "メールアドレス変更の試行回数が上限を超えました。しばらくしてから再度お試しください。")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to generate random token: %w", err), http.StatusInternalServerError)
	}
	plaintextToken := base64.URLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(plaintextToken))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	expiresAt := time.Now().Add(30 * time.Minute)

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ChangeEmail: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteEmailChangeTokensByUserID(ctx, userID); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to delete existing tokens: %w", err), http.StatusInternalServerError)
	}

	if err := qtx.CreateEmailChangeToken(ctx, &queries.CreateEmailChangeTokenParams{
		UserID:    userID,
		NewEmail:  req.NewEmail,
		TokenHash: hashedTokenStr,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to create token: %w", err), http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		actorDBUser, err := qtx.GetUserByID(ctx, userID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangeEmail: failed to get user details for audit log: %w", err), http.StatusInternalServerError)
		}

		tenantID := libctx.GetTenantID(ctx)
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actorDBUser.Name,
			"actor_email": actorDBUser.Email,
			"new_email":   req.NewEmail,
		})

		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceUser),
			Action:       string(auditlog.ActionEmailChanged),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangeEmail: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to commit transaction: %w", err), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("ChangeEmail: failed to marshal SendGrid request: %w", err), http.StatusInternalServerError)
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	response, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ChangeEmail: SendGrid API call failed: %w", err), http.StatusInternalServerError)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errlib.NewError(fmt.Errorf("ChangeEmail: SendGrid API returned error status code %d. Body: %s", response.StatusCode, response.Body), http.StatusInternalServerError)
	}

	return nil
}
