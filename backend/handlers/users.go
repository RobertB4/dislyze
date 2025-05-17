package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"lugia/lib/errors"
	"lugia/lib/middleware"
	"lugia/queries"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsersHandler struct {
	dbConn *pgxpool.Pool
	q      *queries.Queries
}

func NewUsersHandler(dbConn *pgxpool.Pool, q *queries.Queries) *UsersHandler {
	return &UsersHandler{
		dbConn: dbConn,
		q:      q,
	}
}

func (h *UsersHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	rawTenantID, ok := r.Context().Value(middleware.TenantIDKey).(pgtype.UUID)
	if !ok {
		appErr := errors.New(fmt.Errorf("GetUsers: tenant ID not found or invalid type in context"), "", http.StatusUnauthorized)
		errors.LogError(appErr)
		http.Error(w, "Invalid credentials.", http.StatusUnauthorized)
		return
	}

	dbUsers, err := h.q.GetUsersByTenantID(r.Context(), rawTenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]User{})
			return
		}
		appErr := errors.New(err, "Failed to retrieve user list.", http.StatusInternalServerError)
		errors.LogError(appErr)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	responseUsers := make([]User, len(dbUsers))
	for i, dbUser := range dbUsers {
		userIDStr := ""
		if dbUser.ID.Valid {
			userIDStr = dbUser.ID.String()
		} else {
			// This case should ideally not happen for a User's ID (Primary Key).
			// Log an error if it does. userIDStr will remain "".
			errDetail := fmt.Errorf("retrieved user record with invalid/NULL ID (email for context: %s)", dbUser.Email)
			appErr := errors.New(errDetail, "", http.StatusInternalServerError)
			errors.LogError(appErr)
		}

		mappedUser := User{
			ID:        userIDStr,
			Email:     dbUser.Email,
			Role:      dbUser.Role,
			CreatedAt: dbUser.CreatedAt.Time,
			UpdatedAt: dbUser.UpdatedAt.Time,
		}
		if dbUser.Name.Valid {
			mappedUser.Name = dbUser.Name.String
		}
		if dbUser.Status.Valid {
			mappedUser.Status = dbUser.Status.String
		}
		responseUsers[i] = mappedUser
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseUsers)
}
