import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";

test.describe("SSO Login", () => {
	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test("should successfully login via SSO with sso1@sso.test", async ({ page }) => {
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Verify we're on the SSO login page
		await expect(page.getByTestId("sso-login-heading")).toContainText("SSOでログイン");

		// Enter SSO user email
		await page.locator("#email").fill(TestUsersData.sso_1.email);
		await page.getByTestId("sso-login-submit-button").click();

		// Wait for redirect to Keycloak login page
		await page.waitForURL(/.*mock-keycloak.*/, { timeout: 10000 });

		// Fill in Keycloak credentials
		// Keycloak standard login form uses 'username' and 'password' fields
		await page.locator("#username").fill(TestUsersData.sso_1.email);
		await page.locator("#password").fill(TestUsersData.sso_1.plainTextPassword);

		// Submit Keycloak login form
		await page.locator('input[type="submit"][name="login"]').click();

		// Wait for redirect back to our app and successful login
		await page.waitForURL("/", { timeout: 10000 });

		// Verify user is logged in by checking for navigation elements
		await expect(page.getByTestId("navigation-signout")).toBeVisible({ timeout: 5000 });
	});
});
