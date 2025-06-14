package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/pagination"
	"dislyze/jirachi/responder"
	"lugia/lib/search"
	"lugia/queries"
)

type UserRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UserInfo struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	Roles     []UserRole `json:"roles"`
}

type GetUsersResponse struct {
	Users      []UserInfo                    `json:"users"`
	Pagination pagination.PaginationMetadata `json:"pagination"`
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

	usersWithRoles, err := h.q.GetUsersWithRoles(ctx, &queries.GetUsersWithRolesParams{
		TenantID:    tenantID,
		SearchTerm:  searchTerm,
		LimitCount:  paginationParams.Limit,
		OffsetCount: paginationParams.Offset,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)
			response := &GetUsersResponse{
				Users:      []UserInfo{},
				Pagination: paginationMetadata,
			}
			return response, nil
		}
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to get users with roles: %w", err), http.StatusInternalServerError, "")
	}

	var userOrder []string
	userMap := make(map[string]*UserInfo)
	for _, row := range usersWithRoles {
		userID := row.ID.String()

		if _, exists := userMap[userID]; !exists {
			userOrder = append(userOrder, userID)
			userMap[userID] = &UserInfo{
				ID:        userID,
				Email:     row.Email,
				Name:      row.Name,
				Status:    row.Status,
				CreatedAt: row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				Roles:     []UserRole{},
			}
		}

		role := UserRole{
			ID:          row.RoleID.String(),
			Name:        row.RoleName.String,
			Description: row.RoleDescription.String,
		}
		userMap[userID].Roles = append(userMap[userID].Roles, role)
	}

	userInfos := make([]UserInfo, len(userOrder))
	for i, userID := range userOrder {
		userInfos[i] = *userMap[userID]
	}

	paginationMetadata := pagination.CalculateMetadata(paginationParams.Page, paginationParams.Limit, totalCount)

	response := &GetUsersResponse{
		Users:      userInfos,
		Pagination: paginationMetadata,
	}

	return response, nil
}
