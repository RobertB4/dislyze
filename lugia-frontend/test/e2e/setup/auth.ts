import { Page } from "@playwright/test";
import { TestUser } from "./seed";

/**
 * Helper function to login as a specific user
 * @param page The Playwright Page instance
 * @param user The test user data object
 */
export async function logInAs(page: Page, user: TestUser): Promise<void> {
	await page.goto("/auth/login");
	await page.locator("#email").fill(user.email);
	await page.locator("#password").fill(user.plainTextPassword);
	await page.getByTestId("login-submit-button").click();

	// Wait for login to complete and verify redirect to home page
	await page.waitForURL("/");
}
