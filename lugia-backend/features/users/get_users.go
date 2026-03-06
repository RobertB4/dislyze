// Feature doc: docs/features/user-management.md
package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
	"lugia/lib/pagination"
	"lugia/queries"

	"github.com/danielgtaylor/huma/v2"
)

var GetUsersOp = huma.Operation{
	OperationID: "get-users",
	Method:      http.MethodGet,
	Path:        "/users",
}

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
	Roles     []UserRole `json:"roles" nullable:"false"`
}

type GetUsersResponse struct {
	Users      []UserInfo                    `json:"users" nullable:"false"`
	Pagination pagination.PaginationMetadata `json:"pagination"`
}

type GetUsersInput struct {
	Page   int    `query:"page" default:"1" minimum:"1"`
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"100"`
	Search string `query:"search" maxLength:"100"`
}

type GetUsersOutput struct {
	Body GetUsersResponse
}

func (h *UsersHandler) GetUsers(ctx context.Context, input *GetUsersInput) (*GetUsersOutput, error) {
	tenantID := libctx.GetTenantID(ctx)

	limit := int32(input.Limit)
	offset := int32((input.Page - 1) * input.Limit)

	paginationParams := pagination.QueryParams{
		Page:   input.Page,
		Limit:  limit,
		Offset: offset,
	}

	response, err := h.getUsers(ctx, tenantID, paginationParams, input.Search)
	if err != nil {
		return nil, humautil.MapError(err)
	}
	return &GetUsersOutput{Body: *response}, nil
}

func (h *UsersHandler) getUsers(ctx context.Context, tenantID pgtype.UUID, paginationParams pagination.QueryParams, searchTerm string) (*GetUsersResponse, error) {
	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetUsers: tenant not found %s: %w", tenantID.String(), err), http.StatusUnauthorized, "")
		}
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	var enterpriseFeatures authz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to unmarshal features config for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	totalCount, err := h.q.CountUsersByTenantID(ctx, &queries.CountUsersByTenantIDParams{
		TenantID: tenantID,
		Column2:  searchTerm,
	})
	if err != nil {
		return nil, errlib.New(fmt.Errorf("GetUsers: failed to count users: %w", err), http.StatusInternalServerError, "")
	}

	usersWithRoles, err := h.q.GetUsersWithRolesRespectingRBAC(ctx, &queries.GetUsersWithRolesRespectingRBACParams{
		TenantID:    tenantID,
		SearchTerm:  searchTerm,
		LimitCount:  paginationParams.Limit,
		OffsetCount: paginationParams.Offset,
		RbacEnabled: enterpriseFeatures.RBAC.Enabled,
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
			Name:        row.RoleName,
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
