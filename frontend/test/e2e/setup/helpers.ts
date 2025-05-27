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

const pool = new Pool(dbConfig);

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
		const seedFilePath = "../database/seed.sql";

		console.log("Starting database reset (running delete.sql)...");
		await executeSqlFile(deleteFilePath);
		console.log("Database reset (delete.sql executed).");

		console.log("Starting database seeding...");
		await executeSqlFile(seedFilePath);
		console.log("Database seeded.");
		await pool.end();
	} catch (error) {
		console.error("Critical error during database reset/seed process:", error);
		await pool.end().catch(console.error);
		throw error;
	}
}
