package users

import (
	"lugia/lib/pagination"
	"lugia/queries"
	"lugia/queries_pregeneration"
)

type ChangeEmailRequestBody struct {
	NewEmail string `json:"new_email"`
}

type ChangePasswordRequestBody struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	NewPasswordConfirm string `json:"new_password_confirm"`
}

type ChangeTenantNameRequestBody struct {
	Name string `json:"name"`
}

type MeResponse struct {
	TenantName string `json:"tenant_name"`
	TenantPlan string `json:"tenant_plan"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	UserName   string `json:"user_name"`
	UserRole   string `json:"user_role"`
}

type GetUsersResponse struct {
	Users      []*queries.GetUsersByTenantIDRow `json:"users"`
	Pagination pagination.PaginationMetadata    `json:"pagination"`
}

type InviteUserRequestBody struct {
	Email string                         `json:"email"`
	Name  string                         `json:"name"`
	Role  queries_pregeneration.UserRole `json:"role"`
}

type UpdateMeRequestBody struct {
	Name string `json:"name"`
}

type UpdateUserRoleRequestBody struct {
	Role queries_pregeneration.UserRole `json:"role"`
}
