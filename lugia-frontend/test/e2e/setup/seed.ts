export const TestUsersData = {
	alpha_admin: {
		email: "alpha_admin@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000001",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha Admin",
		status: "active"
	},
	alpha_editor: {
		email: "alpha_editor@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000002",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Alpha Editor",
		status: "active"
	},
	pending_editor_valid_token: {
		email: "pending_editor_valid_token@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000003",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Pending Editor Valid Token",
		status: "pending_verification"
	},
	suspended_editor: {
		email: "suspended_editor@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000004",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Suspended Editor",
		status: "suspended"
	},
	pending_editor_for_rate_limit_test: {
		email: "pending_editor_for_rate_limit_test@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000005",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "Pending Editor Rate Limit Test",
		status: "pending_verification"
	},
	pending_editor_tenant_A_for_x_tenant_test: {
		email: "pending_editor_tenant_A_for_x_tenant_test@example.com",
		plainTextPassword: "password",
		userID: "b0000000-0000-0000-0000-000000000006",
		tenantID: "a0000000-0000-0000-0000-000000000001",
		name: "PendingXT Editor",
		status: "pending_verification"
	},
	// Tenant Beta User
	beta_admin: {
		email: "beta_admin@example.com",
		plainTextPassword: "password123",
		userID: "b0000000-0000-0000-0000-000000000007",
		tenantID: "a0000000-0000-0000-0000-000000000002",
		name: "Beta Admin",
		status: "active"
	}
} as const;

export type TestUserKey = keyof typeof TestUsersData;
export type TestUser = (typeof TestUsersData)[TestUserKey];

export const TestTenantsData = {
	tenant_alpha: {
		id: "a0000000-0000-0000-0000-000000000001",
		name: "Tenant Alpha"
	},
	tenant_beta: {
		id: "a0000000-0000-0000-0000-000000000002",
		name: "Tenant Beta"
	}
} as const;

export type TestTenantKey = keyof typeof TestTenantsData;
export type TestTenant = (typeof TestTenantsData)[TestTenantKey];
