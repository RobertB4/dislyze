package setup

// TestUsersData provides easy access to details of users seeded by seed.sql
var TestUsersData = map[string]struct {
	Email             string
	PlainTextPassword string
	UserID            string
	TenantID          string
	Name              string
	Role              string
	Status            string
}{
	"alpha_admin": {
		Email:             "alpha_admin@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000001",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Alpha Admin",
		Role:              "admin",
		Status:            "active",
	},
	"alpha_editor": {
		Email:             "alpha_editor@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000002",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Alpha Editor",
		Role:              "editor",
		Status:            "active",
	},
	"pending_editor_valid_token": {
		Email:             "pending_editor_valid_token@example.com",
		PlainTextPassword: "password",
		UserID:            "b0000000-0000-0000-0000-000000000003",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Pending Editor Valid Token",
		Role:              "editor",
		Status:            "pending_verification",
	},
	"suspended_editor": {
		Email:             "suspended_editor@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000004",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Suspended Editor",
		Role:              "editor",
		Status:            "suspended",
	},
	"pending_editor_for_rate_limit_test": {
		Email:             "pending_editor_for_rate_limit_test@example.com",
		PlainTextPassword: "password",
		UserID:            "b0000000-0000-0000-0000-000000000005",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Pending Editor Rate Limit Test",
		Role:              "editor",
		Status:            "pending_verification",
	},
	"pending_editor_tenant_A_for_x_tenant_test": {
		Email:             "pending_editor_tenant_A_for_x_tenant_test@example.com",
		PlainTextPassword: "password",
		UserID:            "b0000000-0000-0000-0000-000000000006",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "PendingXT Editor",
		Role:              "editor",
		Status:            "pending_verification",
	},
	"beta_admin": {
		Email:             "beta_admin@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000007",
		TenantID:          "a0000000-0000-0000-0000-000000000002",
		Name:              "Beta Admin",
		Role:              "admin",
		Status:            "active",
	},
}
