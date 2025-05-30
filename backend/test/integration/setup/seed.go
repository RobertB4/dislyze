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
	"pending_user_valid_token": {
		Email:             "pending_user_valid_token@example.com",
		PlainTextPassword: "password", // Initial password before activation
		UserID:            "b0000000-0000-0000-0000-000000000003",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Pending User Valid Token",
		Role:              "editor",
		Status:            "pending_verification",
	},
	"suspended_user": {
		Email:             "suspended_user@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000004",
		TenantID:          "a0000000-0000-0000-0000-000000000001",
		Name:              "Suspended User",
		Role:              "editor",
		Status:            "suspended",
	},
	"beta_admin": {
		Email:             "beta_admin@example.com",
		PlainTextPassword: "password123",
		UserID:            "b0000000-0000-0000-0000-000000000005",
		TenantID:          "a0000000-0000-0000-0000-000000000002",
		Name:              "Beta Admin",
		Role:              "admin",
		Status:            "active",
	},
}
