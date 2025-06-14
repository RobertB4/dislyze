package setup

type TenantTestData struct {
	ID                 string
	Name               string
	EnterpriseFeatures map[string]interface{}
}

type RoleTestData struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	IsDefault   bool
}

type PermissionTestData struct {
	ID          string
	Resource    string
	Action      string
	Description string
}

type UserTestData struct {
	Email             string
	PlainTextPassword string
	UserID            string
	TenantID          string
	Name              string
	Status            string
}

type UserRoleTestData struct {
	UserID   string
	RoleID   string
	TenantID string
}

var TestTenantsData = map[string]TenantTestData{
	"internal": {
		ID:   "33333333-3333-3333-3333-333333333333",
		Name: "内部株式会社",
		EnterpriseFeatures: map[string]interface{}{
			"rbac": map[string]interface{}{"enabled": true},
		},
	},
}

var TestPermissionsData = map[string]PermissionTestData{
	"tenant_view": {
		ID:          "98ca3771-c1d3-46fc-8ec6-04d53ea38322",
		Resource:    "tenant",
		Action:      "view",
		Description: "テナント情報の閲覧",
	},
	"tenant_edit": {
		ID:          "6e95ed87-f380-41fe-bc5b-f8af002345a4",
		Resource:    "tenant",
		Action:      "edit",
		Description: "テナント情報の編集",
	},
	"users_view": {
		ID:          "3a52c807-ddcb-4044-8682-658e04800a8e",
		Resource:    "users",
		Action:      "view",
		Description: "ユーザー一覧の閲覧",
	},
	"users_edit": {
		ID:          "db994eda-6ff7-4ae5-a675-3abe735ce9cc",
		Resource:    "users",
		Action:      "edit",
		Description: "ユーザーの編集",
	},
	"roles_view": {
		ID:          "44b8962d-5dc5-490e-8469-03078668dd52",
		Resource:    "roles",
		Action:      "view",
		Description: "ロール一覧の閲覧",
	},
	"roles_edit": {
		ID:          "cccf277b-5fd5-4f1d-b763-ebf69973e5b7",
		Resource:    "roles",
		Action:      "edit",
		Description: "ロールの編集",
	},
}

var TestRolesData = map[string]RoleTestData{
	"internal_admin": {
		ID:          "22222222-3333-4444-5555-666666666666",
		TenantID:    "33333333-3333-3333-3333-333333333333",
		Name:        "管理者",
		Description: "すべての機能にアクセス可能",
		IsDefault:   true,
	},
	"internal_editor": {
		ID:          "33333333-4444-5555-6666-777777777777",
		TenantID:    "33333333-3333-3333-3333-333333333333",
		Name:        "編集者",
		Description: "ユーザー管理以外の編集権限",
		IsDefault:   true,
	},
	"internal_viewer": {
		ID:          "44444444-5555-6666-7777-888888888888",
		TenantID:    "33333333-3333-3333-3333-333333333333",
		Name:        "閲覧者",
		Description: "閲覧権限のみ",
		IsDefault:   true,
	},
}

var TestUsersData = map[string]UserTestData{
	"internal_1": {
		Email:             "internal1@localhost.com",
		PlainTextPassword: "1234567890",
		UserID:            "c0000000-0000-0000-0000-000000000001",
		TenantID:          "33333333-3333-3333-3333-333333333333",
		Name:              "管理 太郎",
		Status:            "active",
	},
	"internal_2": {
		Email:             "internal2@localhost.com",
		PlainTextPassword: "1234567890",
		UserID:            "c0000000-0000-0000-0000-000000000002",
		TenantID:          "33333333-3333-3333-3333-333333333333",
		Name:              "運営 花子",
		Status:            "active",
	},
	"enterprise_1": {
		Email:             "enterprise1@localhost.com",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000001",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "田中 太郎",
		Status:            "active",
	},
}

var TestUserRolesData = map[string]UserRoleTestData{
	"internal_1_admin": {
		UserID:   "c0000000-0000-0000-0000-000000000001",
		RoleID:   "22222222-3333-4444-5555-666666666666",
		TenantID: "33333333-3333-3333-3333-333333333333",
	},
	"internal_2_editor": {
		UserID:   "c0000000-0000-0000-0000-000000000002",
		RoleID:   "33333333-4444-5555-6666-777777777777",
		TenantID: "33333333-3333-3333-3333-333333333333",
	},
}
