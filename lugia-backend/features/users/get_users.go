package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/pagination"
	"lugia/lib/responder"
	"lugia/lib/search"
	"lugia/queries"
)

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
