import fs from "node:fs/promises";
import path from "node:path";
import { Pool, type PoolConfig } from "pg";

const dbConfig: PoolConfig = {
	user: process.env.DB_USER,
	password: process.env.DB_PASSWORD,
	host: process.env.DB_HOST,
	database: process.env.DB_NAME,
	port: parseInt(process.env.DB_PORT as string, 10),
	ssl: process.env.DB_SSL_MODE === "require"
};

export const pool = new Pool(dbConfig);

pool.on("error", (err) => {
	console.error("Unexpected error on idle client in pg pool", err);
	process.exit(-1);
});

async function executeSqlFile(filePath: string): Promise<void> {
	const resolvedPath = path.resolve(process.cwd(), filePath);
	console.log(`Executing SQL file: ${resolvedPath}`);
	try {
		const sql = await fs.readFile(resolvedPath, { encoding: "utf8" });
		const client = await pool.connect();
		try {
			await client.query(sql);
			console.log(`Successfully executed SQL file: ${resolvedPath}`);
		} finally {
			client.release();
		}
	} catch (error) {
		console.error(`Error executing SQL file ${resolvedPath}:`, error);
		throw error;
	}
}

export async function resetAndSeedDatabase(): Promise<void> {
	try {
		const deleteFilePath = "../database/delete.sql";
		const seedFilePath = "../database/seed_localhost.sql";

		console.log("Starting database reset (running delete.sql)...");
		await executeSqlFile(deleteFilePath);
		console.log("Database reset (delete.sql executed).");

		console.log("Starting database seeding...");
		await executeSqlFile(seedFilePath);
		console.log("Database seeded.");
	} catch (error) {
		console.error("Critical error during database reset/seed process:", error);
		throw error;
	}
}

/**
 * Creates a new tenant with RBAC feature disabled via signup API
 * Returns credentials for the created admin user
 */
export async function createTenant(): Promise<{
	email: string;
	password: string;
	tenantId: string;
	userId: string;
}> {
	const baseUrl = process.env.PLAYWRIGHT_BASE_URL || "http://localhost:23000";
	const timestamp = Date.now();

	const signupData = {
		email: `rbac_disabled_admin_${timestamp}@example.com`,
		password: "password123",
		password_confirm: "password123",
		company_name: `RBAC Disabled Tenant ${timestamp}`,
		user_name: `RBAC Disabled Admin ${timestamp}`
	};

	const response = await fetch(`${baseUrl}/api/auth/signup`, {
		method: "POST",
		headers: {
			"Content-Type": "application/json"
		},
		body: JSON.stringify(signupData)
	});

	if (!response.ok) {
		throw new Error(`Signup failed: ${response.status} ${response.statusText}`);
	}

	// Get tenant and user IDs from database
	const client = await pool.connect();
	try {
		const result = await client.query(
			"SELECT u.id as user_id, u.tenant_id FROM users u WHERE u.email = $1",
			[signupData.email]
		);

		if (result.rows.length === 0) {
			throw new Error("Created user not found in database");
		}

		return {
			email: signupData.email,
			password: signupData.password,
			tenantId: result.rows[0].tenant_id,
			userId: result.rows[0].user_id
		};
	} finally {
		client.release();
	}
}

/**
 * Enables RBAC feature for a specific tenant
 */
export async function enableRBACForTenant(tenantId: string): Promise<void> {
	const client = await pool.connect();
	try {
		await client.query(
			`UPDATE tenants SET enterprise_features = '{"rbac": {"enabled": true}}'::jsonb WHERE id = $1`,
			[tenantId]
		);
	} finally {
		client.release();
	}
}

/**
 * Disables RBAC feature for a specific tenant
 */
export async function disableRBACForTenant(tenantId: string): Promise<void> {
	const client = await pool.connect();
	try {
		await client.query(
			`UPDATE tenants SET enterprise_features = '{"rbac": {"enabled": false}}'::jsonb WHERE id = $1`,
			[tenantId]
		);
	} finally {
		client.release();
	}
}
