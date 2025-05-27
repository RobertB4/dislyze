import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";

test.describe("Auth - Signup Page", () => {
	const signupURL = "/auth/signup";
	const loginURL = "/auth/login";
	const uniqueUserEmail = `testuser_${Date.now()}@example.com`;

	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.beforeEach(async ({ page }) => {
		await page.goto(signupURL);
	});

	test("should load the page correctly and link to login", async ({ page }) => {
		await expect(page.getByTestId("signup-heading")).toBeVisible();
		await expect(page.locator("#company_name")).toBeVisible();
		await expect(page.locator("#user_name")).toBeVisible();
		await expect(page.locator("#email")).toBeVisible();
		await expect(page.locator("#password")).toBeVisible();
		await expect(page.locator("#password_confirm")).toBeVisible();
		await expect(page.getByTestId("signup-submit-button")).toBeVisible();

		await page.getByTestId("login-link").click();
		await expect(page).toHaveURL(loginURL);
	});

	test("should display error when email already exists", async ({ page }) => {
		const headingLocator = page.getByTestId("signup-heading");
		await expect(headingLocator).toBeVisible({ timeout: 15000 });

		const companyNameLocator = page.locator("#company_name");
		await expect(companyNameLocator).toBeVisible({ timeout: 10000 });

		await companyNameLocator.fill("Test Company Existing Email");
		await page.locator("#user_name").fill("Test User Existing");
		await page.locator("#email").fill(TestUsersData.alpha_admin.email);
		await page.locator("#password").fill("password123");
		await page.locator("#password_confirm").fill("password123");

		await page.getByTestId("signup-submit-button").click();

		await expect(page).toHaveURL(signupURL);

		const toastMessage = page.getByTestId("toast-0");
		await expect(toastMessage).toBeVisible({ timeout: 10000 });
		await expect(toastMessage).toContainText("このメールアドレスは既に使用されています。");
	});

	test("should display validation errors for all required fields if empty", async ({ page }) => {
		await page.getByTestId("signup-submit-button").click();

		const companyNameErrorLocator = page.getByTestId("company_name-error");
		await expect(companyNameErrorLocator).toContainText("会社名は必須です", { timeout: 7000 });

		await expect(page.getByTestId("user_name-error")).toContainText("氏名は必須です");
		await expect(page.getByTestId("email-error")).toContainText("メールアドレスは必須です");
		await expect(page.getByTestId("password-error")).toContainText("パスワードは必須です");
		await expect(page.getByTestId("password_confirm-error")).toContainText(
			"パスワードを確認してください"
		);
	});

	test("should display validation error for invalid email format", async ({ page }) => {
		await page.locator("#email").fill("invalidemail");
		await page.getByTestId("signup-submit-button").click();
		await expect(page.getByTestId("email-error")).toContainText(
			"メールアドレスの形式が正しくありません"
		);
	});

	test("should display validation error for password too short", async ({ page }) => {
		await page.locator("#password").fill("1234567");
		await page.getByTestId("signup-submit-button").click();
		await expect(page.getByTestId("password-error")).toContainText(
			"パスワードは8文字以上である必要があります"
		);
	});

	test("should display validation error when passwords do not match", async ({ page }) => {
		await page.locator("#password").fill("password123");
		await page.locator("#password_confirm").fill("password456");
		await page.getByTestId("signup-submit-button").click();
		await expect(page.getByTestId("password_confirm-error")).toContainText(
			"パスワードが一致しません"
		);
	});

	test("should allow successful user registration with unique email", async ({ page, baseURL }) => {
		await page.locator("#company_name").fill("New Test Corp Proxy");
		await page.locator("#user_name").fill("New Test User Proxy");
		await page.locator("#email").fill(uniqueUserEmail);
		await page.locator("#password").fill("validPassword123");
		await page.locator("#password_confirm").fill("validPassword123");

		await page.getByTestId("signup-submit-button").click();

		// Wait for the /api/me call that happens after successful registration and login
		// This ensures cookies are set and the frontend app has processed the login.
		await page.waitForResponse(
			(response) => response.url().includes("/api/me") && response.status() === 200,
			{ timeout: 15000 }
		);

		// The application should redirect to the root ('/') on successful signup and /me authentication
		const expectedHomePageURL = baseURL && baseURL.endsWith("/") ? baseURL : `${baseURL}/`;
		await expect(page).toHaveURL(expectedHomePageURL, { timeout: 15000 });

		await expect(page.locator('[data-testid^="toast-"]')).not.toBeVisible({ timeout: 2000 });
	});
});
