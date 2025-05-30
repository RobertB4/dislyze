export const TestUsersData = {
	alpha_admin: {
		email: "alpha_admin@example.com",
		plainTextPassword: "password123", // As per comment in seed.sql
		userID: "b0000000-0000-0000-0000-000000000001",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha Admin",
		role: "admin",
		status: "active"
	},
	alpha_user: {
		email: "alpha_user@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000002",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha User",
		role: "editor",
		status: "active"
	},
	pending_editor_valid_token: {
		email: "pending_user_valid_token@example.com",
		plainTextPassword: "password123", // seed.sql comment says 'password123', Go file says 'password'. Assuming seed.sql is source of truth for hash.
		userID: "b0000000-0000-0000-0000-000000000003",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Pending User Valid Token",
		role: "editor",
		status: "pending_verification" // As per seed.sql
	},
	suspended_user: {
		email: "suspended_user@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000004",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Suspended User",
		role: "editor",
		status: "suspended"
	},
	beta_admin: {
		email: "beta_admin@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000005",
		tenantID: "a0000000-0000-0000-0000-000000000002",
		name: "Beta Admin",
		role: "admin",
		status: "active"
	}
} as const;

export type TestUserKey = keyof typeof TestUsersData;
export type TestUser = (typeof TestUsersData)[TestUserKey];

// Example Tenant Data (if useful for tests)
export const TestTenantsData = {
	tenant_alpha: {
		id: "a0000000-0000-0000-0000-000000000001",
		name: "Tenant Alpha",
		plan: "basic"
	},
	tenant_beta: {
		id: "a0000000-0000-0000-0000-000000000002",
		name: "Tenant Beta",
		plan: "basic"
	}
} as const;

export type TestTenantKey = keyof typeof TestTenantsData;
export type TestTenant = (typeof TestTenantsData)[TestTenantKey];
