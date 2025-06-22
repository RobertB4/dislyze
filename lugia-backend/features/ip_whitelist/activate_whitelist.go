package ip_whitelist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	jirachiAuthz "dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"dislyze/jirachi/sendgridlib"
	"lugia/lib/iputils"
	"lugia/lib/jwt"
	"lugia/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"
)

type ActivateWhitelistRequestBody struct {
	Force bool `json:"force,omitempty"`
}

type ActivateWhitelistResponse struct {
	UserIP string `json:"user_ip,omitempty"`
}

func (h *IPWhitelistHandler) ActivateWhitelist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ActivateWhitelistRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("ActivateWhitelist: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("ActivateWhitelist: failed to close request body: %w", err))
		}
	}()

	userIP, err := h.activateWhitelist(ctx, req, r)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	if userIP != "" {
		response := ActivateWhitelistResponse{
			UserIP: userIP,
		}
		responder.RespondWithJSON(w, http.StatusOK, response)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *IPWhitelistHandler) activateWhitelist(ctx context.Context, req ActivateWhitelistRequestBody, r *http.Request) (string, error) {
	tenantID := libctx.GetTenantID(ctx)
	userID := libctx.GetUserID(ctx)
	userIP := iputils.ExtractClientIP(r)

	if !req.Force {
		isSafe, err := h.validateActivationSafety(ctx, tenantID, userIP)
		if err != nil {
			return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to validate activation safety: %w", err), http.StatusInternalServerError, "")
		}

		if !isSafe {
			return userIP, nil
		}
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to get tenant: %w", err), http.StatusInternalServerError, "")
	}

	if len(tenant.EnterpriseFeatures) == 0 {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: tenant %s has no enterprise features configured", tenantID.String()), http.StatusInternalServerError, "")
	}

	var currentFeatures jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &currentFeatures); err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to parse enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	currentFeatures.IPWhitelist.Active = true

	updatedFeaturesJSON, err := json.Marshal(currentFeatures)
	if err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to marshal enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ActivateWhitelist: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.UpdateTenantEnterpriseFeatures(ctx, &queries.UpdateTenantEnterpriseFeaturesParams{
		EnterpriseFeatures: updatedFeaturesJSON,
		ID:                 tenantID,
	})
	if err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to update tenant enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	if req.Force {
		err = h.createEmergencyTokenAndSendEmail(ctx, qtx, tenantID, userID)
		if err != nil {
			return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to create emergency token and send email: %w", err), http.StatusInternalServerError, "")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", errlib.New(fmt.Errorf("ActivateWhitelist: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return "", nil
}

func (h *IPWhitelistHandler) validateActivationSafety(ctx context.Context, tenantID pgtype.UUID, userIP string) (bool, error) {
	existingCIDRs, err := h.q.GetTenantIPWhitelistCIDRs(ctx, tenantID)
	if err != nil {
		return false, err
	}

	if len(existingCIDRs) == 0 {
		return false, nil
	}

	isAllowed, err := iputils.IsIPInCIDRList(userIP, existingCIDRs)
	if err != nil {
		return false, err
	}

	return isAllowed, nil
}

func (h *IPWhitelistHandler) createEmergencyTokenAndSendEmail(ctx context.Context, qtx *queries.Queries, tenantID, userID pgtype.UUID) error {
	emergencyToken, jti, err := jwt.GenerateEmergencyToken(userID, tenantID, []byte(h.env.IPWhitelistEmergencyJWTSecret))
	if err != nil {
		return fmt.Errorf("failed to generate emergency token: %w", err)
	}
	_, err = qtx.CreateIPWhitelistEmergencyToken(ctx, jti)
	if err != nil {
		return fmt.Errorf("failed to create emergency token record: %w", err)
	}

	user, err := qtx.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user for emergency email: %w", err)
	}

	err = h.sendEmergencyEmail(user.Email, user.Name, emergencyToken)
	if err != nil {
		return fmt.Errorf("failed to send emergency email: %w", err)
	}

	return nil
}

func (h *IPWhitelistHandler) sendEmergencyEmail(email, name, token string) error {
	subject := "【緊急】IPアクセス制限の解除用リンク"

	emergencyLink := fmt.Sprintf("%s/ip-whitelist/emergency-deactivate?token=%s", h.env.FrontendURL, token)

	plainTextContent := fmt.Sprintf("%s様。\n\nIPアクセス制限が有効化されました。万が一アクセスできなくなってしまった場合は、下記のリンクからアクセス制限を解除することができます。\n\n解除用リンク（30分間有効）：\n%s\n\n※このメールは自動送信されています。ご不明な点がございましたら、サポートチームまでご連絡ください。",
		name, emergencyLink)

	htmlContent := fmt.Sprintf("<p>%s様</p><p>IPアクセス制限が有効化されました。万が一アクセスできなくなってしまった場合は、下記のリンクからアクセス制限を解除することができます。</p><p><a href=\"%s\" style=\"background-color: #dc3545; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; display: inline-block;\">アクセス制限を解除する</a></p><p><small>※このリンクは30分間有効です</small></p><p><small>※このメールは自動送信されています。ご不明な点がございましたら、サポートチームまでご連絡ください。</small></p>",
		name, emergencyLink)

	sgMailBody := sendgridlib.SendGridMailRequestBody{
		Personalizations: []sendgridlib.SendGridPersonalization{
			{
				To:      []sendgridlib.SendGridEmailAddress{{Email: email, Name: name}},
				Subject: subject,
			},
		},
		From:    sendgridlib.SendGridEmailAddress{Email: sendgridlib.SendGridFromEmail, Name: sendgridlib.SendGridFromName},
		Content: []sendgridlib.SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		return fmt.Errorf("failed to marshal SendGrid request body: %w", err)
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		return fmt.Errorf("SendGrid API call failed: %w", err)
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		return fmt.Errorf("SendGrid returned error status code %d. Body: %s", sgResponse.StatusCode, sgResponse.Body)
	}

	return nil
}
