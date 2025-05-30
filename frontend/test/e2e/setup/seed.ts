export const TestUsersData = {
	alpha_admin: {
		email: "alpha_admin@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000001",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha Admin",
		role: "admin",
		status: "active"
	},
	alpha_editor: {
		email: "alpha_editor@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000002",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha Editor",
		role: "editor",
		status: "active"
	},
	pending_user_valid_token: {
		email: "pending_user_valid_token@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000003",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Pending User Valid Token",
		role: "editor",
		status: "pending_verification"
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
	pending_user_for_rate_limit_test: {
		email: "pending_user_for_rate_limit_test@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000005",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Pending User Rate Limit Test",
		role: "editor",
		status: "pending_verification"
	},
	pending_user_tenant_A_for_x_tenant_test: {
		email: "pending_user_tenant_A_for_x_tenant_test@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000006",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "PendingXT User",
		role: "editor",
		status: "pending_verification"
	},
	// Tenant Beta User
	beta_admin: {
		email: "beta_admin@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000007",
		tenantID: "a0000000-0000-0000-0000-000000000002",
		name: "Beta Admin",
		role: "admin",
		status: "active"
	}
} as const;

export type TestUserKey = keyof typeof TestUsersData;
export type TestUser = (typeof TestUsersData)[TestUserKey];

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
