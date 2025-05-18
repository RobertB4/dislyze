package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sendgrid/sendgrid-go"
	"golang.org/x/crypto/bcrypt"

	"lugia/lib/config"
	libctx "lugia/lib/ctx"
	"lugia/lib/errors"
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
	if r.Role != "admin" && r.Role != "user" {
		return fmt.Errorf("role is invalid, must be 'admin' or 'user'")
	}
	return nil
}

type InviteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type UsersHandler struct {
	dbConn *pgxpool.Pool
	q      *queries.Queries
	env    *config.Env
}

func NewUsersHandler(dbConn *pgxpool.Pool, q *queries.Queries, env *config.Env) *UsersHandler {
	return &UsersHandler{
		dbConn: dbConn,
		q:      q,
		env:    env,
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
		errors.LogError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rawTenantID := libctx.GetTenantID(ctx)

	_, err := h.q.GetUserByEmail(ctx, req.Email)
	if err == nil {
		errResp := InviteUserResponse{Error: "指定されたメールアドレスは既に使用されています。"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedInitialPassword, err := bcrypt.GenerateFromPassword([]byte(h.env.InitialPW), bcrypt.DefaultCost)
	if err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) && !errors.Is(rbErr, sql.ErrTxDone) {
			errors.LogError(fmt.Errorf("InviteUser: failed to rollback transaction: %w", rbErr))
		}
	}()

	qtx := h.q.WithTx(tx)

	err = qtx.InviteUserToTenant(ctx, &queries.InviteUserToTenantParams{
		TenantID:     rawTenantID,
		Email:        req.Email,
		PasswordHash: string(hashedInitialPassword),
		Name:         req.Name,
		Role:         req.Role,
		Status:       "pending_verification",
	})
	if err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	subject := "dislyzeへのご招待"
	// TODO: Generate a secure, unique, and time-limited token for the invitation link.
	// For now, using a placeholder. The actual token should be stored and verified.
	invitationToken := "PLACEHOLDER_INVITATION_TOKEN_" + req.Email                              // Replace with real token
	invitationLink := fmt.Sprintf("https://%s/accept-invite?token=%s", r.Host, invitationToken) // Assuming r.Host gives a usable domain for the link

	plainTextContent := fmt.Sprintf("%s 様、\\n\\n%s dislyzeに招待されました。\\n\\n以下のリンクをクリックして登録を完了してください。\\n%s\\n\\nこのメールにお心当たりがない場合は、無視してください。", req.Name, sendGridFromName, invitationLink)
	htmlContent := fmt.Sprintf(`<p>%s 様</p>
	<p>%s dislyzeに招待されました。</p>
	<p>以下のリンクをクリックして登録を完了してください。</p>
	<p><a href="%s">登録を完了する</a></p>
	<p>このメールにお心当たりがない場合は、無視してください。</p>`, req.Name, sendGridFromName, invitationLink)

	sgMailBody := SendGridMailRequestBody{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmailAddress{
					{Email: req.Email, Name: req.Name},
				},
				Subject: subject,
			},
		},
		From: SendGridEmailAddress{Email: sendGridFromEmail, Name: sendGridFromName},
		Content: []SendGridContent{
			{Type: "text/plain", Value: plainTextContent},
			{Type: "text/html", Value: htmlContent},
		},
	}

	bodyBytes, err := json.Marshal(sgMailBody)
	if err != nil {
		errors.LogError(fmt.Errorf("failed to marshal SendGrid request body: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendgridRequest := sendgrid.GetRequest(h.env.SendgridAPIKey, "/v3/mail/send", h.env.SendgridAPIUrl)
	sendgridRequest.Method = "POST"
	sendgridRequest.Body = bodyBytes
	response, err := sendgrid.API(sendgridRequest)
	if err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errors.LogError(fmt.Errorf("SendGrid Sendgrid API return error status code: %d, Body: %s", response.StatusCode, response.Body))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		errors.LogError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	successResp := InviteUserResponse{
		Success: true,
		Message: "招待メールを送信しました。",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(successResp)
}
