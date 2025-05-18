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

var (
	ErrInvalidUserDataFromDB = fmt.Errorf("invalid user data retrieved from database")
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
			Role:      dbUser.Role,
			Status:    dbUser.Status,
			CreatedAt: dbUser.CreatedAt.Time,
			UpdatedAt: dbUser.UpdatedAt.Time,
		}
		if dbUser.Name.Valid {
			mappedUser.Name = dbUser.Name.String
		}
		responseUsers[i] = mappedUser
	}
	return responseUsers, nil
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

	responseUsers, mapErr := mapDBUsersToResponse(dbUsers)
	if mapErr != nil {
		appErr := errors.New(mapErr, "Internal server error during data processing.", http.StatusInternalServerError)
		errors.LogError(appErr)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(responseUsers)
}
