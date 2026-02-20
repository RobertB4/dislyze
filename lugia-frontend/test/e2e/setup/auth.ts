import type { Page } from "@playwright/test";
import type { TestUser } from "./seed";

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

/**
 * Helper function to logout the current user by clicking the logout button
 * @param page The Playwright Page instance
 */
export async function logOut(page: Page): Promise<void> {
	// Navigate to home page and wait for it to load completely
	await page.goto("/");

	// Wait for the layout to be ready by checking for navigation elements
	await page.waitForSelector(
		'[data-testid="navigation-signout"], [data-testid="navigation-signout-mobile"]',
		{
			timeout: 2000
		}
	);

	// Try desktop logout button first (visible on larger screens)
	const desktopLogoutButton = page.getByTestId("navigation-signout");
	const mobileLogoutButton = page.getByTestId("navigation-signout-mobile");

	if (await desktopLogoutButton.isVisible()) {
		await desktopLogoutButton.click();
	} else if (await mobileLogoutButton.isVisible()) {
		await mobileLogoutButton.click();
	} else {
		// Fallback: clear cookies if buttons aren't visible
		await page.context().clearCookies();
		return;
	}

	// Wait for logout to complete and redirect to login page
	await page.waitForURL(/.*\/auth\/login.*/);
}
