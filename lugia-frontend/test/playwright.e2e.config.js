import { defineConfig, devices } from "@playwright/test";

const headed = process.env.PLAYWRIGHT_HEADED ? process.env.PLAYWRIGHT_HEADED === "true" : true;

export default defineConfig({
	testDir: "./e2e", // Relative to this config file
	globalTeardown: "./e2e/setup/globalTeardown.ts",
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI === "true" ? 1 : 0,
	workers: 1,
	timeout: 10 * 1000, // 10 seconds
	reporter: [["html", { outputFolder: "./test-results", open: "never" }], ["list"]],
	use: {
		baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:23000",
		trace: "on-first-retry",
		headless: !headed
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] }
		}
	]
});
