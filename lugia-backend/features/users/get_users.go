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
	Users      []*queries.GetUsersByTenantIDRow `json:"users"`
	Pagination pagination.PaginationMetadata    `json:"pagination"`
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

	users, err := h.q.GetUsersByTenantID(ctx, &queries.GetUsersByTenantIDParams{
		TenantID: tenantID,
		Column2:  searchTerm,
		Limit:    paginationParams.Limit,
		Offset:   paginationParams.Offset,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)
			response := &GetUsersResponse{
				Users:      []*queries.GetUsersByTenantIDRow{},
				Pagination: paginationMetadata,
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to get users: %w", err), http.StatusInternalServerError, "")
	}

	paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)

	response := &GetUsersResponse{
		Users:      users,
		Pagination: paginationMetadata,
	}

	return response, nil
}
