import { pool } from "$lugia-test/e2e/setup/helpers";

async function globalTeardown() {
	console.log("Global teardown: Closing PostgreSQL connection pool...");
	await pool.end();
	console.log("PostgreSQL connection pool closed.");
}

export default globalTeardown;
