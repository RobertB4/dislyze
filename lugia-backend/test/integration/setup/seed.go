package setup

// TenantTestData defines the structure for tenant test data
type TenantTestData struct {
	ID                 string
	Name               string
	EnterpriseFeatures map[string]interface{}
}

// RoleTestData defines the structure for role test data
type RoleTestData struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	IsDefault   bool
}

// PermissionTestData defines the structure for permission test data
type PermissionTestData struct {
	ID          string
	Resource    string
	Action      string
	Description string
}

// UserTestData defines the structure for individual test user data
type UserTestData struct {
	Email             string
	PlainTextPassword string
	UserID            string
	TenantID          string
	Name              string
	Status            string
}

// UserRoleTestData defines the structure for user role assignments
type UserRoleTestData struct {
	UserID   string
	RoleID   string
	TenantID string
}

// InvitationTokenTestData defines the structure for invitation token test data
type InvitationTokenTestData struct {
	ID            string
	TokenHash     string
	TenantID      string
	UserID        string
	UnhashedToken string
}

// TestTenantsData provides easy access to tenant data
var TestTenantsData = map[string]TenantTestData{
	"enterprise": {
		ID:   "11111111-1111-1111-1111-111111111111",
		Name: "エンタープライズ株式会社",
		EnterpriseFeatures: map[string]interface{}{
			"rbac": map[string]interface{}{"enabled": true},
		},
	},
	"smb": {
		ID:   "22222222-2222-2222-2222-222222222222",
		Name: "SMB株式会社",
		EnterpriseFeatures: map[string]interface{}{
			"rbac": map[string]interface{}{"enabled": false},
		},
	},
	"internal": {
		ID:   "33333333-3333-3333-3333-333333333333",
		Name: "内部株式会社",
		EnterpriseFeatures: map[string]interface{}{
			"rbac": map[string]interface{}{"enabled": true},
		},
	},
}

// TestPermissionsData provides easy access to permission data
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
	"ip_whitelist_view": {
		ID:          "f1a8b2c3-d4e5-f6a7-b8c9-d0e1f2a3b4c5",
		Resource:    "ip_whitelist",
		Action:      "view",
		Description: "IP制限画面の閲覧",
	},
	"ip_whitelist_edit": {
		ID:          "a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4",
		Resource:    "ip_whitelist",
		Action:      "edit",
		Description: "IP制限画面の編集",
	},
}

// TestRolesData provides easy access to role data
var TestRolesData = map[string]RoleTestData{
	"enterprise_admin": {
		ID:          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		TenantID:    "11111111-1111-1111-1111-111111111111",
		Name:        "管理者",
		Description: "すべての機能にアクセス可能",
		IsDefault:   true,
	},
	"enterprise_editor": {
		ID:          "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID:    "11111111-1111-1111-1111-111111111111",
		Name:        "編集者",
		Description: "ユーザー管理以外の編集権限",
		IsDefault:   true,
	},
	"enterprise_viewer": {
		ID:          "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID:    "11111111-1111-1111-1111-111111111111",
		Name:        "閲覧者",
		Description: "閲覧権限のみ",
		IsDefault:   true,
	},
	"enterprise_user_manager": {
		ID:          "dddddddd-dddd-dddd-dddd-dddddddddddd",
		TenantID:    "11111111-1111-1111-1111-111111111111",
		Name:        "ユーザー管理者",
		Description: "ユーザーの閲覧と編集権限を持つ",
		IsDefault:   false,
	},
	"smb_admin": {
		ID:          "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		TenantID:    "22222222-2222-2222-2222-222222222222",
		Name:        "管理者",
		Description: "すべての機能にアクセス可能",
		IsDefault:   true,
	},
	"smb_editor": {
		ID:          "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID:    "22222222-2222-2222-2222-222222222222",
		Name:        "編集者",
		Description: "ユーザー管理以外の編集権限",
		IsDefault:   true,
	},
	"smb_viewer": {
		ID:          "11111111-2222-3333-4444-555555555555",
		TenantID:    "22222222-2222-2222-2222-222222222222",
		Name:        "閲覧者",
		Description: "閲覧権限のみ",
		IsDefault:   true,
	},
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

// TestUsersData provides easy access to details of users seeded by seed.sql
var TestUsersData = map[string]UserTestData{
	// Internal User (is_internal_user = true, employee user)
	"internal_user_enterprise": {
		Email:             "11111111-1111-1111-1111-111111111111@internal.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000000",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "内部ユーザー",
		Status:            "active",
	},

	// Enterprise Users (first 20 only)
	"enterprise_1": {
		Email:             "enterprise1@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000001",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "田中 太郎",
		Status:            "active",
	},
	"enterprise_2": {
		Email:             "enterprise2@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000002",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "佐藤 花子",
		Status:            "active",
	},
	"enterprise_3": {
		Email:             "enterprise3@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000003",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "鈴木 一郎",
		Status:            "active",
	},
	"enterprise_4": {
		Email:             "enterprise4@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000004",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "高橋 美咲",
		Status:            "active",
	},
	"enterprise_5": {
		Email:             "enterprise5@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000005",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "伊藤 健太",
		Status:            "active",
	},
	"enterprise_6": {
		Email:             "enterprise6@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000006",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "渡辺 真由美",
		Status:            "active",
	},
	"enterprise_7": {
		Email:             "enterprise7@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000007",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "山本 慎太郎",
		Status:            "active",
	},
	"enterprise_8": {
		Email:             "enterprise8@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000008",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "中村 由美",
		Status:            "active",
	},
	"enterprise_9": {
		Email:             "enterprise9@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000009",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "小林 大輔",
		Status:            "active",
	},
	"enterprise_10": {
		Email:             "enterprise10@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000010",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "加藤 恵",
		Status:            "active",
	},
	"enterprise_11": {
		Email:             "enterprise11@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000011",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "吉田 雄二",
		Status:            "pending_verification",
	},
	"enterprise_12": {
		Email:             "enterprise12@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000012",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "福田 あかり",
		Status:            "pending_verification",
	},
	"enterprise_13": {
		Email:             "enterprise13@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000013",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "森 拓也",
		Status:            "pending_verification",
	},
	"enterprise_14": {
		Email:             "enterprise14@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000014",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "清水 愛",
		Status:            "pending_verification",
	},
	"enterprise_15": {
		Email:             "enterprise15@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000015",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "三浦 翔太",
		Status:            "pending_verification",
	},
	"enterprise_16": {
		Email:             "enterprise16@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000016",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "橋本 詩織",
		Status:            "suspended",
	},
	"enterprise_17": {
		Email:             "enterprise17@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000017",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "岡田 浩一",
		Status:            "suspended",
	},
	"enterprise_18": {
		Email:             "enterprise18@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000018",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "前田 麻衣",
		Status:            "suspended",
	},
	"enterprise_19": {
		Email:             "enterprise19@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000019",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "長谷川 竜也",
		Status:            "suspended",
	},
	"enterprise_20": {
		Email:             "enterprise20@enterprise.test",
		PlainTextPassword: "1234567890",
		UserID:            "a0000000-0000-0000-0000-000000000020",
		TenantID:          "11111111-1111-1111-1111-111111111111",
		Name:              "石川 結衣",
		Status:            "suspended",
	},

	// SMB Users (all 10)
	"smb_1": {
		Email:             "smb1@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000001",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "青木 直人",
		Status:            "active",
	},
	"smb_2": {
		Email:             "smb2@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000002",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "松井 知子",
		Status:            "active",
	},
	"smb_3": {
		Email:             "smb3@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000003",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "大石 悟",
		Status:            "active",
	},
	"smb_4": {
		Email:             "smb4@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000004",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "野口 裕子",
		Status:            "active",
	},
	"smb_5": {
		Email:             "smb5@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000005",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "森 達夫",
		Status:            "active",
	},
	"smb_6": {
		Email:             "smb6@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000006",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "菊池 典子",
		Status:            "active",
	},
	"smb_7": {
		Email:             "smb7@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000007",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "中山 拓",
		Status:            "active",
	},
	"smb_8": {
		Email:             "smb8@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000008",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "水野 彩",
		Status:            "active",
	},
	"smb_9": {
		Email:             "smb9@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000009",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "柴田 勝",
		Status:            "active",
	},
	"smb_10": {
		Email:             "smb10@smb.test",
		PlainTextPassword: "1234567890",
		UserID:            "b0000000-0000-0000-0000-000000000010",
		TenantID:          "22222222-2222-2222-2222-222222222222",
		Name:              "坂田 恵里",
		Status:            "active",
	},

	// Internal Users (all 2)
	"internal_1": {
		Email:             "internal1@internal.test",
		PlainTextPassword: "1234567890",
		UserID:            "c0000000-0000-0000-0000-000000000001",
		TenantID:          "33333333-3333-3333-3333-333333333333",
		Name:              "管理 太郎",
		Status:            "active",
	},
	"internal_2": {
		Email:             "internal2@internal.test",
		PlainTextPassword: "1234567890",
		UserID:            "c0000000-0000-0000-0000-000000000002",
		TenantID:          "33333333-3333-3333-3333-333333333333",
		Name:              "運営 花子",
		Status:            "active",
	},
}

// TestUserRolesData provides easy access to user role assignments
var TestUserRolesData = map[string]UserRoleTestData{
	// Enterprise tenant role assignments
	"enterprise_1_admin": {
		UserID:   "a0000000-0000-0000-0000-000000000001",
		RoleID:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_2_editor": {
		UserID:   "a0000000-0000-0000-0000-000000000002",
		RoleID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_3_editor": {
		UserID:   "a0000000-0000-0000-0000-000000000003",
		RoleID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_4_editor": {
		UserID:   "a0000000-0000-0000-0000-000000000004",
		RoleID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_5_editor": {
		UserID:   "a0000000-0000-0000-0000-000000000005",
		RoleID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_6_editor": {
		UserID:   "a0000000-0000-0000-0000-000000000006",
		RoleID:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_7_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000007",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_8_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000008",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_9_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000009",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_10_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000010",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_11_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000011",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_12_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000012",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_13_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000013",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_14_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000014",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_15_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000015",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_16_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000016",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_17_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000017",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_18_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000018",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_19_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000019",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},
	"enterprise_20_viewer": {
		UserID:   "a0000000-0000-0000-0000-000000000020",
		RoleID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID: "11111111-1111-1111-1111-111111111111",
	},

	// SMB tenant role assignments
	"smb_1_admin": {
		UserID:   "b0000000-0000-0000-0000-000000000001",
		RoleID:   "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_2_editor": {
		UserID:   "b0000000-0000-0000-0000-000000000002",
		RoleID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_3_editor": {
		UserID:   "b0000000-0000-0000-0000-000000000003",
		RoleID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_4_editor": {
		UserID:   "b0000000-0000-0000-0000-000000000004",
		RoleID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_5_editor": {
		UserID:   "b0000000-0000-0000-0000-000000000005",
		RoleID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_6_editor": {
		UserID:   "b0000000-0000-0000-0000-000000000006",
		RoleID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_7_viewer": {
		UserID:   "b0000000-0000-0000-0000-000000000007",
		RoleID:   "11111111-2222-3333-4444-555555555555",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_8_viewer": {
		UserID:   "b0000000-0000-0000-0000-000000000008",
		RoleID:   "11111111-2222-3333-4444-555555555555",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_9_viewer": {
		UserID:   "b0000000-0000-0000-0000-000000000009",
		RoleID:   "11111111-2222-3333-4444-555555555555",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},
	"smb_10_viewer": {
		UserID:   "b0000000-0000-0000-0000-000000000010",
		RoleID:   "11111111-2222-3333-4444-555555555555",
		TenantID: "22222222-2222-2222-2222-222222222222",
	},

	// Internal tenant role assignments
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

// TestInvitationTokensData provides easy access to invitation token data
var TestInvitationTokensData = map[string]InvitationTokenTestData{
	"enterprise_11_token": {
		ID:            "d0000000-0000-0000-0000-000000000001",
		TokenHash:     "3874d22b39c01882df8ee09c464ebc7441293d6e295299d35e26a8ec12f68a3d",
		TenantID:      "11111111-1111-1111-1111-111111111111",
		UserID:        "a0000000-0000-0000-0000-000000000011",
		UnhashedToken: "26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ",
	},
	"enterprise_12_token": {
		ID:            "d0000000-0000-0000-0000-000000000002",
		TokenHash:     "7de910a7f03bb39e3c24375b8f34d787d8af449706904edc83986224de97c163",
		TenantID:      "11111111-1111-1111-1111-111111111111",
		UserID:        "a0000000-0000-0000-0000-000000000012",
		UnhashedToken: "MO-Cw4btd5KDj1TK16yxNo-zkFtkyjyjOlqUZ5AFWYA=",
	},
	"enterprise_13_token": {
		ID:            "d0000000-0000-0000-0000-000000000003",
		TokenHash:     "e6418e6d62b63b8a0a0eac30ba45797ac9273590c2110efc8ed5453711310473",
		TenantID:      "11111111-1111-1111-1111-111111111111",
		UserID:        "a0000000-0000-0000-0000-000000000013",
		UnhashedToken: "CM-rBsZ3PDoIenm_Od4pRdUMAcIgqUWlqs3rMSvmrk0=",
	},
	"enterprise_10_active_user_token": {
		ID:            "d0000000-0000-0000-0000-000000000004",
		TokenHash:     "7261975581a789f841dca6d6261cb3a6ab06bdec4a81e987ca4f8865a3c3fe67",
		TenantID:      "11111111-1111-1111-1111-111111111111",
		UserID:        "a0000000-0000-0000-0000-000000000010",
		UnhashedToken: "accept-invite-active-user-token-for-testing",
	},
	"enterprise_14_expired_token": {
		ID:            "d0000000-0000-0000-0000-000000000005",
		TokenHash:     "1689934ddd1d942277310ce36b363be5bd6201523f348d2dda35ebce74643db3",
		TenantID:      "11111111-1111-1111-1111-111111111111",
		UserID:        "a0000000-0000-0000-0000-000000000014",
		UnhashedToken: "accept-invite-expired-token-for-testing",
	},
}
