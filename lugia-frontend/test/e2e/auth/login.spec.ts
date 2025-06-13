import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";

test.describe("Auth - Login Page", () => {
	const loginURL = "/auth/login";

	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.beforeEach(async ({ page }) => {
		await page.goto(loginURL);
	});

	test("should load the page correctly and link to signup", async ({ page }) => {
		await expect(page.getByTestId("login-heading")).toBeVisible();
		await expect(page.locator("#email")).toBeVisible();
		await expect(page.locator("#password")).toBeVisible();
		await expect(page.getByTestId("login-submit-button")).toBeVisible();
		await expect(page.getByTestId("forgot-password-link")).toBeVisible();

		await page.getByTestId("signup-link").click();
		await expect(page).toHaveURL("/auth/signup");
	});

	test("should display validation errors for empty fields", async ({ page }) => {
		await page.getByTestId("login-submit-button").click();

		await expect(page.getByTestId("email-error")).toContainText("メールアドレスは必須です", {
			timeout: 7000
		});
		await expect(page.getByTestId("password-error")).toContainText("パスワードは必須です", {
			timeout: 7000
		});
	});

	test("should display validation error for invalid email format", async ({ page }) => {
		await page.locator("#email").fill("invalidemail");
		await page.getByTestId("login-submit-button").click();
		await expect(page.getByTestId("email-error")).toContainText(
			"メールアドレスの形式が正しくありません",
			{ timeout: 7000 }
		);
	});

	test("should display error for non-existent email", async ({ page }) => {
		await page.locator("#email").fill("nonexistent@example.com");
		await page.locator("#password").fill("password123");
		await page.getByTestId("login-submit-button").click();

		const toastMessage = page.getByTestId("toast-0");
		await expect(toastMessage).toBeVisible({ timeout: 10000 });
		await expect(toastMessage).toContainText("メールアドレスまたはパスワードが正しくありません");
		await expect(page).toHaveURL(loginURL);
	});

	test("should display error for incorrect password", async ({ page }) => {
		await page.locator("#email").fill(TestUsersData.enterprise_1.email);
		await page.locator("#password").fill("wrongpassword");
		await page.getByTestId("login-submit-button").click();

		const toastMessage = page.getByTestId("toast-0");
		await expect(toastMessage).toBeVisible({ timeout: 10000 });
		await expect(toastMessage).toContainText("メールアドレスまたはパスワードが正しくありません");
		await expect(page).toHaveURL(loginURL);
	});

	test("should allow successful login with correct credentials", async ({ page, baseURL }) => {
		await page.locator("#email").fill(TestUsersData.enterprise_1.email);
		await page.locator("#password").fill(TestUsersData.enterprise_1.plainTextPassword);
		await page.getByTestId("login-submit-button").click();

		await page.waitForResponse(
			(response) => response.url().includes("/api/me") && response.status() === 200,
			{ timeout: 15000 }
		);

		await expect(page).toHaveURL(`/`, { timeout: 15000 });

		await expect(page.locator('[data-testid^="toast-"]')).not.toBeVisible({ timeout: 2000 });
	});

	test("should display error for pending verification account", async ({ page }) => {
		await page.locator("#email").fill(TestUsersData.enterprise_11.email);
		await page.locator("#password").fill(TestUsersData.enterprise_11.plainTextPassword);
		await page.getByTestId("login-submit-button").click();

		const toastMessage = page.getByTestId("toast-0");
		await expect(toastMessage).toBeVisible({ timeout: 10000 });
		await expect(toastMessage).toContainText(
			"アカウントが有効化されていません。招待メールを確認し、登録を完了してください。"
		);
		await expect(page).toHaveURL(loginURL);
	});

	test("should display error for suspended account", async ({ page }) => {
		await page.locator("#email").fill(TestUsersData.enterprise_16.email);
		await page.locator("#password").fill(TestUsersData.enterprise_16.plainTextPassword);
		await page.getByTestId("login-submit-button").click();

		const toastMessage = page.getByTestId("toast-0");
		await expect(toastMessage).toBeVisible({ timeout: 10000 });
		await expect(toastMessage).toContainText(
			"アカウントが停止されています。サポートにお問い合わせください。"
		);
		await expect(page).toHaveURL(loginURL);
	});

	test("should navigate to forgot password page when link is clicked", async ({ page }) => {
		await page.getByTestId("forgot-password-link").click();
		await expect(page).toHaveURL("/auth/forgot-password");
	});

	test("should allow successful login by pressing Enter in password field", async ({ page }) => {
		await page.locator("#email").fill(TestUsersData.enterprise_1.email);
		const passwordInput = page.locator("#password");
		await passwordInput.fill(TestUsersData.enterprise_1.plainTextPassword);
		await passwordInput.press("Enter");

		await page.waitForResponse(
			(response) => response.url().includes("/api/me") && response.status() === 200,
			{ timeout: 15000 }
		);

		await expect(page).toHaveURL(`/`, { timeout: 15000 });
		await expect(page.locator('[data-testid^="toast-"]')).not.toBeVisible({ timeout: 2000 });
	});
});
