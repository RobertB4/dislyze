import { test, expect } from "@playwright/test";

test.describe("Frontend Health Check", () => {
	test("should be able to fetch the root page with status 200", async ({ request, baseURL }) => {
		console.log(`Attempting to fetch base URL (from config): ${baseURL}, path: /`);

		const response = await request.get("/");
		const status = response.status();

		console.log(`Response status from GET /: ${status}`);

		// Log headers if not 200, for more debug info
		if (status !== 200) {
			console.log(`Response headers: ${JSON.stringify(response.headers(), null, 2)}`);
			// Optionally log body if it's small and text-based for 403s
			// const body = await response.text();
			// console.log(`Response body: ${body.substring(0, 500)}...`);
		}

		expect(status).toBe(200);
	});
});
