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
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sendgrid/sendgrid-go"
	"golang.org/x/crypto/bcrypt"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/lib/sendgridlib"
	"lugia/queries"
	"lugia/queries_pregeneration"
)

type InviteUserRequest struct {
	Email string                         `json:"email"`
	Name  string                         `json:"name"`
	Role  queries_pregeneration.UserRole `json:"role"`
}

func (r *InviteUserRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.Name = strings.TrimSpace(r.Name)
	r.Role = queries_pregeneration.UserRole(strings.TrimSpace(strings.ToLower(r.Role.String())))

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.ContainsRune(r.Email, '@') {
		return fmt.Errorf("email is invalid")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required and cannot be only whitespace")
	}
	if r.Role == "" {
		return fmt.Errorf("role is required")
	}
	if r.Role != "admin" && r.Role != "editor" {
		return fmt.Errorf("role is invalid, must be 'admin' or 'editor'")
	}
	return nil
}

func (h *UsersHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("InviteUser: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	rawTenantID := libctx.GetTenantID(ctx)
	inviterUserID := libctx.GetUserID(ctx)

	inviterDBUser, err := h.q.GetUserByID(ctx, inviterUserID)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to get inviter's user details for UserID %s: %w", inviterUserID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	_, err = h.q.GetUserByEmail(ctx, req.Email)
	if err == nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: attempt to invite existing email: %s", req.Email), http.StatusConflict, "このメールアドレスは既に使用されています。")
		responder.RespondWithError(w, appErr)
		return
	}
	if !errlib.Is(err, pgx.ErrNoRows) {
		appErr := errlib.New(fmt.Errorf("InviteUser: GetUserByEmail failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	hashedInitialPassword, err := bcrypt.GenerateFromPassword([]byte(h.env.InitialPW), bcrypt.DefaultCost)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to hash initial password: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("InviteUser: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	createdUserID, err := qtx.InviteUserToTenant(ctx, &queries.InviteUserToTenantParams{
		TenantID:     rawTenantID,
		Email:        req.Email,
		PasswordHash: string(hashedInitialPassword),
		Name:         req.Name,
		Role:         req.Role,
		Status:       "pending_verification",
	})
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: InviteUserToTenant failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to generate random bytes for invitation token: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	plaintextToken := base64.URLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(plaintextToken))
	hashedTokenStr := fmt.Sprintf("%x", hash[:])

	expiresAt := time.Now().Add(48 * time.Hour) // 2 days

	_, err = qtx.CreateInvitationToken(ctx, &queries.CreateInvitationTokenParams{
		TokenHash: hashedTokenStr,
		TenantID:  rawTenantID,
		UserID:    createdUserID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: CreateInvitationToken failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	subject := fmt.Sprintf("%sさんから%s様へのdislyzeへのご招待", inviterDBUser.Name, req.Name)
	invitationLink := fmt.Sprintf("%s/auth/accept-invite?token=%s&inviter_name=%s&invited_email=%s",
		h.env.FrontendURL,
		plaintextToken,
		url.QueryEscape(inviterDBUser.Name),
		url.QueryEscape(req.Email))

	plainTextContent := fmt.Sprintf("%s様、\n\n%sさんがあなたをdislyzeに招待しています。\n\n以下のリンクをクリックして登録を完了してください。\n%s\n\nこのメールにお心当たりがない場合は、無視してください。", req.Name, inviterDBUser.Name, invitationLink)
	htmlContent := fmt.Sprintf(`<p>%s様</p>
	<p>%sさんがあなたをdislyzeに招待しています。</p>
	<p>以下のリンクをクリックして登録を完了してください。</p>
	<p><a href="%s">登録を完了する</a></p>
	<p>このメールにお心当たりがない場合は、無視してください。</p>`, req.Name, inviterDBUser.Name, invitationLink)

	sgMailBody := sendgridlib.SendGridMailRequestBody{
		Personalizations: []sendgridlib.SendGridPersonalization{
			{
				To:      []sendgridlib.SendGridEmailAddress{{Email: req.Email, Name: req.Name}},
				Subject: subject,
			},
		},
		From:    sendgridlib.SendGridEmailAddress{Email: sendgridlib.SendGridFromEmail, Name: sendgridlib.SendGridFromName},
		Content: []sendgridlib.SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to marshal SendGrid request body: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	response, err := sendgrid.API(sendgridRequest)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: SendGrid API call failed: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		appErr := errlib.New(fmt.Errorf("InviteUser: SendGrid API returned error status code: %d, Body: %s", response.StatusCode, response.Body), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		appErr := errlib.New(fmt.Errorf("InviteUser: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}
