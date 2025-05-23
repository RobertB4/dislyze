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

	"lugia/lib/config"
	libctx "lugia/lib/ctx"
	"lugia/lib/errors"
	"lugia/lib/ratelimit"
	"lugia/queries"
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
		if errors.Is(err, pgx.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]User{})
			return
		}
		errors.LogError(err)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	responseUsers, mapErr := mapDBUsersToResponse(dbUsers)
	if mapErr != nil {
		errors.LogError(err)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseUsers)
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
		errors.LogError(fmt.Errorf("InviteUser: failed to decode request: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		errors.LogError(fmt.Errorf("InviteUser: validation failed: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawTenantID := libctx.GetTenantID(ctx)
	inviterUserID := libctx.GetUserID(ctx)

	inviterDBUser, err := h.q.GetUserByID(ctx, inviterUserID)
	if err != nil {
		errors.LogError(fmt.Errorf("InviteUser: failed to get inviter's user details for UserID %s: %w", inviterUserID.String(), err))
	}

	_, err = h.q.GetUserByEmail(ctx, req.Email)
	if err == nil {
		// User found, email already exists
		errors.LogError(fmt.Errorf("InviteUser: attempt to invite existing email: %s", req.Email))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "このメールアドレスは既に使用されています。"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		errors.LogError(fmt.Errorf("InviteUser: GetUserByEmail failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedInitialPassword, err := bcrypt.GenerateFromPassword([]byte(h.env.InitialPW), bcrypt.DefaultCost)
	if err != nil {
		errors.LogError(fmt.Errorf("InviteUser: failed to hash initial password: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		errors.LogError(fmt.Errorf("InviteUser: failed to begin transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) && !errors.Is(rbErr, sql.ErrTxDone) {
			errors.LogError(fmt.Errorf("InviteUser: failed to rollback transaction: %w", rbErr))
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
		errors.LogError(fmt.Errorf("InviteUser: InviteUserToTenant failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		errors.LogError(fmt.Errorf("InviteUser: failed to generate random bytes for invitation token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("InviteUser: CreateInvitationToken failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("InviteUser: failed to marshal SendGrid request body: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	response, err := sendgrid.API(sendgridRequest)
	if err != nil {
		errors.LogError(fmt.Errorf("InviteUser: SendGrid API call failed: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errors.LogError(fmt.Errorf("InviteUser: SendGrid API returned error status code: %d, Body: %s", response.StatusCode, response.Body))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		errors.LogError(fmt.Errorf("InviteUser: failed to commit transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *UsersHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := chi.URLParam(r, "userID")

	if !h.resendInviteRateLimiter.Allow(targetUserIDStr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "招待メールの再送信は、ユーザーごとに5分間に1回のみ可能です。しばらくしてから再度お試しください。"})
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: invalid target userID format '%s': %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: Implement proper authorization: check if current user is admin of the target user's tenant.
	// TODO: Implement specific rate limiting for this endpoint (e.g., per targetUserID).

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	invokerDBUser, err := h.q.GetUserByID(ctx, invokerUserID)
	if err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: failed to get invoker's user details for UserID %s: %w", invokerUserID.String(), err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if invokerDBUser == nil {
		errors.LogError(fmt.Errorf("ResendInvite: invoker user not found for UserID %s", invokerUserID.String()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(fmt.Errorf("ResendInvite: target user with ID %s not found: %w", targetUserIDStr, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		errors.LogError(fmt.Errorf("ResendInvite: failed to get target user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if invokerTenantID != targetDBUser.TenantID {
		errors.LogError(fmt.Errorf("ResendInvite: invoker %s (tenant %s) attempting to resend invite for user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if targetDBUser.Status != "pending_verification" {
		errors.LogError(fmt.Errorf("ResendInvite: target user %s status is '%s', expected 'pending_verification'", targetUserIDStr, targetDBUser.Status))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: failed to begin transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) && !errors.Is(rbErr, sql.ErrTxDone) {
			errors.LogError(fmt.Errorf("ResendInvite: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteInvitationTokensByUserIDAndTenantID(ctx, &queries.DeleteInvitationTokensByUserIDAndTenantIDParams{
		UserID:   targetUserID,
		TenantID: targetDBUser.TenantID,
	}); err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: failed to delete existing invitation tokens for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: failed to generate random bytes for invitation token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("ResendInvite: CreateInvitationToken failed for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
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
		errors.LogError(fmt.Errorf("ResendInvite: failed to marshal SendGrid request body for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	sgResponse, err := sendgrid.API(sendgridRequest)
	if err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: SendGrid API call failed for user %s: %w.", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sgResponse.StatusCode < 200 || sgResponse.StatusCode >= 300 {
		errors.LogError(fmt.Errorf("ResendInvite: SendGrid API returned error status code %d for user %s. Body: %s.", sgResponse.StatusCode, targetUserIDStr, sgResponse.Body))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		errors.LogError(fmt.Errorf("ResendInvite: failed to commit transaction for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetUserIDStr := chi.URLParam(r, "userID")

	if !h.deleteUserRateLimiter.Allow(targetUserIDStr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "ユーザー削除の操作は制限されています。しばらくしてから再度お試しください。"})
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: invalid target userID format '%s': %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: Implement proper authorization: check if current user is admin of the target user's tenant.
	// For now, we assume the authenticated user has permission to delete users.

	invokerUserID := libctx.GetUserID(ctx)
	invokerTenantID := libctx.GetTenantID(ctx)

	targetDBUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errors.LogError(fmt.Errorf("DeleteUser: target user with ID %s not found: %w", targetUserIDStr, err))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		errors.LogError(fmt.Errorf("DeleteUser: failed to get target user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if invokerTenantID != targetDBUser.TenantID {
		errors.LogError(fmt.Errorf("DeleteUser: invoker %s (tenant %s) attempting to delete user %s (tenant %s) in different tenant", invokerUserID.String(), invokerTenantID.String(), targetUserID.String(), targetDBUser.TenantID.String()))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if invokerUserID == targetUserID {
		errors.LogError(fmt.Errorf("DeleteUser: user %s attempting to delete themselves", invokerUserID.String()))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "自分自身を削除することはできません。"})
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: failed to begin transaction: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) && !errors.Is(rbErr, sql.ErrTxDone) {
			errors.LogError(fmt.Errorf("DeleteUser: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	if err := qtx.DeleteInvitationTokensByUserIDAndTenantID(ctx, &queries.DeleteInvitationTokensByUserIDAndTenantIDParams{
		UserID:   targetUserID,
		TenantID: targetDBUser.TenantID,
	}); err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: failed to delete invitation tokens for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := qtx.DeleteRefreshTokensByUserID(ctx, targetUserID); err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: failed to delete refresh tokens for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := qtx.DeleteUser(ctx, targetUserID); err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: failed to delete user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		errors.LogError(fmt.Errorf("DeleteUser: failed to commit transaction for user %s: %w", targetUserIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
