package users

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/pagination"
	"lugia/lib/responder"
	"lugia/lib/search"
	"lugia/queries"
	"lugia/queries_pregeneration"
)

var (
	ErrInvalidUserDataFromDB = fmt.Errorf("invalid user data retrieved from database")
)

type User struct {
	ID        string                         `json:"id"`
	Email     string                         `json:"email"`
	Name      string                         `json:"name,omitempty"`
	Role      queries_pregeneration.UserRole `json:"role"`
	Status    string                         `json:"status"`
	CreatedAt time.Time                      `json:"created_at"`
	UpdatedAt time.Time                      `json:"updated_at"`
}

type GetUsersResponse struct {
	Users      []User                        `json:"users"`
	Pagination pagination.PaginationMetadata `json:"pagination"`
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

	paginationParams := pagination.CalculatePagination(r)
	searchTerm := search.ValidateSearchTerm(r, 100)

	response, err := h.getUsers(ctx, rawTenantID, paginationParams, searchTerm)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *UsersHandler) getUsers(ctx context.Context, tenantID pgtype.UUID, paginationParams pagination.QueryParams, searchTerm string) (*GetUsersResponse, error) {
	totalCount, err := h.q.CountUsersByTenantID(ctx, &queries.CountUsersByTenantIDParams{
		TenantID: tenantID,
		Column2:  searchTerm,
	})
	if err != nil {
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to count users: %w", err), http.StatusInternalServerError, "")
	}

	dbUsers, err := h.q.GetUsersByTenantID(ctx, &queries.GetUsersByTenantIDParams{
		TenantID: tenantID,
		Column2:  searchTerm,
		Limit:    paginationParams.Limit,
		Offset:   paginationParams.Offset,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)
			response := &GetUsersResponse{
				Users:      []User{},
				Pagination: paginationMetadata,
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to get users: %w", err), http.StatusInternalServerError, "")
	}

	responseUsers, mapErr := mapDBUsersToResponse(dbUsers)
	if mapErr != nil {
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to map users: %w", mapErr), http.StatusInternalServerError, "")
	}

	paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)

	response := &GetUsersResponse{
		Users:      responseUsers,
		Pagination: paginationMetadata,
	}

	return response, nil
}