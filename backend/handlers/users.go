package handlers

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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sendgrid/sendgrid-go"
	"golang.org/x/crypto/bcrypt"

	"dislyze/lib/config"
	libctx "dislyze/lib/ctx"
	"dislyze/lib/errlib"
	"dislyze/lib/ratelimit"
	"dislyze/lib/responder"
	"dislyze/queries"
)

var (
	ErrInvalidUserDataFromDB = fmt.Errorf("invalid user data retrieved from database")
)

const (
	sendGridFromName  = "dislyze"
	sendGridFromEmail = "support@dislyze.com"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InviteUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (r *InviteUserRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.Name = strings.TrimSpace(r.Name)
	r.Role = strings.TrimSpace(strings.ToLower(r.Role))

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

type InviteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type MeResponse struct {
	TenantName string `json:"tenant_name"`
	TenantPlan string `json:"tenant_plan"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	UserName   string `json:"user_name"`
	UserRole   string `json:"user_role"`
}

type UsersHandler struct {
	dbConn                  *pgxpool.Pool
	q                       *queries.Queries
	env                     *config.Env
	resendInviteRateLimiter *ratelimit.RateLimiter
	deleteUserRateLimiter   *ratelimit.RateLimiter
}

func NewUsersHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env, resendInviteRateLimiter *ratelimit.RateLimiter, deleteUserRateLimiter *ratelimit.RateLimiter) *UsersHandler {
	return &UsersHandler{
		dbConn:                  dbConn,
		q:                       q,
		env:                     env,
		resendInviteRateLimiter: resendInviteRateLimiter,
		deleteUserRateLimiter:   deleteUserRateLimiter,
	}
}

func mapDBUsersToResponse(dbUsers []*queries.GetUsersByTenantIDRow) ([]User, error) {
	responseUsers := make([]User, len(dbUsers))
	for i, dbUser := range dbUsers {
		if dbUser == nil {
			// This is highly unexpected if the DB query is correct.
			return nil, fmt.Errorf("%w: encountered nil user record at index %d", ErrInvalidUserDataFromDB, i)
		}
		userIDStr := ""
		if dbUser.ID.Valid {
			userIDStr = dbUser.ID.String()
		} else {
			// This case should ideally not happen for a User's ID (Primary Key).
			return nil, fmt.Errorf("%w: user record with invalid/NULL ID (email for context: %s)", ErrInvalidUserDataFromDB, dbUser.Email)
		}

		mappedUser := User{
			ID:        userIDStr,
			Email:     dbUser.Email,
			Name:      dbUser.Name,
			Role:      dbUser.Role,
			Status:    dbUser.Status,
			CreatedAt: dbUser.CreatedAt.Time,
			UpdatedAt: dbUser.UpdatedAt.Time,
		}
		responseUsers[i] = mappedUser
	}
	return responseUsers, nil
}

func (h *UsersHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawTenantID := libctx.GetTenantID(ctx)

	dbUsers, err := h.q.GetUsersByTenantID(r.Context(), rawTenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			responder.RespondWithJSON(w, http.StatusOK, []User{})
			return
		}
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	responseUsers, mapErr := mapDBUsersToResponse(dbUsers)
	if mapErr != nil {
		appErr := errlib.New(mapErr, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, responseUsers)
}

type SendGridEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SendGridPersonalization struct {
	To      []SendGridEmailAddress `json:"to"`
	Subject string                 `json:"subject"`
}

type SendGridMailRequestBody struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmailAddress      `json:"from"`
	Content          []SendGridContent         `json:"content"`
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

	sgMailBody := SendGridMailRequestBody{
		Personalizations: []SendGridPersonalization{
			{
				To:      []SendGridEmailAddress{{Email: req.Email, Name: req.Name}},
				Subject: subject,
			},
		},
		From:    SendGridEmailAddress{Email: sendGridFromEmail, Name: sendGridFromName},
		Content: []SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
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

	responder.RespondWithJSON(w, http.StatusCreated, map[string]bool{"success": true})
}

func (h *UsersHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := chi.URLParam(r, "userID")

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

	sgMailBody := SendGridMailRequestBody{
		Personalizations: []SendGridPersonalization{
			{
				To:      []SendGridEmailAddress{{Email: targetDBUser.Email, Name: targetDBUser.Name}},
				Subject: subject,
			},
		},
		From:    SendGridEmailAddress{Email: sendGridFromEmail, Name: sendGridFromName},
		Content: []SendGridContent{{Type: "text/plain", Value: plainTextContent}, {Type: "text/html", Value: htmlContent}},
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

	responder.RespondWithJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := chi.URLParam(r, "userID")

	if !h.deleteUserRateLimiter.Allow(targetUserIDStr) {
		appErr := errlib.New(fmt.Errorf("rate limit exceeded for user %s delete", targetUserIDStr), http.StatusTooManyRequests, "ユーザー削除の操作は制限されています。しばらくしてから再度お試しください。")
		responder.RespondWithError(w, appErr)
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: invalid target userID format '%s': %w", targetUserIDStr, err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("DeleteUser: target user with ID %s not found: %w", targetUserIDStr, err), http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to get target user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if invokerTenantID != targetDBUser.TenantID {
		appErr := errlib.New(fmt.Errorf("DeleteUser: invoker %s (tenant %s) attempting to delete user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()), http.StatusForbidden, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if invokerUserID == targetUserID {
		appErr := errlib.New(fmt.Errorf("DeleteUser: user %s attempting to delete themselves", invokerUserID.String()), http.StatusConflict, "自分自身を削除することはできません。")
		responder.RespondWithError(w, appErr)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("DeleteUser: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteInvitationTokensByUserIDAndTenantID(ctx, &queries.DeleteInvitationTokensByUserIDAndTenantIDParams{
		UserID:   targetUserID,
		TenantID: targetDBUser.TenantID,
	}); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to delete invitation tokens for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, targetUserID); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to delete refresh tokens for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := qtx.DeleteUser(ctx, targetUserID); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to delete user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteUser: failed to commit transaction for user %s: %w", targetUserIDStr, err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	responder.RespondWithJSON(w, http.StatusNoContent, nil)
}

func (h *UsersHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	response := MeResponse{
		TenantName: tenant.Name,
		TenantPlan: tenant.Plan,
		UserID:     user.ID.String(),
		Email:      user.Email,
		UserName:   user.Name,
		UserRole:   user.Role,
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}
