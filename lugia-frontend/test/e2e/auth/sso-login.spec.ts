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

		// Verify user is logged in by checking user email in layout
		await expect(page.getByTestId("layout-user-email")).toContainText(TestUsersData.sso_1.email);
	});

	test("should auto-provision new user on first SSO login", async ({ page }) => {
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter email for user that exists in Keycloak but NOT in database
		await page.locator("#email").fill("ssonewuser@sso.test");
		await page.getByTestId("sso-login-submit-button").click();

		// Wait for redirect to Keycloak login page
		await page.waitForURL(/.*mock-keycloak.*/, { timeout: 10000 });

		// Fill in Keycloak credentials (password from realm.json)
		await page.locator("#username").fill("ssonewuser@sso.test");
		await page.locator("#password").fill("1234567890");

		// Submit Keycloak login form
		await page.locator('input[type="submit"][name="login"]').click();

		// Wait for redirect back to our app and successful login
		await page.waitForURL("/", { timeout: 10000 });

		// Verify user is logged in by checking user email in layout (user was auto-provisioned)
		await expect(page.getByTestId("layout-user-email")).toContainText("ssonewuser@sso.test");
	});

	test("should activate pending_verification user on SSO login", async ({ page }) => {
		// sso2@sso.test exists in database with status='pending_verification' (from seed data)
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter email for pending user
		await page.locator("#email").fill(TestUsersData.sso_2.email);
		await page.getByTestId("sso-login-submit-button").click();

		// Wait for redirect to Keycloak login page
		await page.waitForURL(/.*mock-keycloak.*/, { timeout: 10000 });

		// Fill in Keycloak credentials
		await page.locator("#username").fill(TestUsersData.sso_2.email);
		await page.locator("#password").fill(TestUsersData.sso_2.plainTextPassword);

		// Submit Keycloak login form
		await page.locator('input[type="submit"][name="login"]').click();

		// Wait for redirect back to our app and successful login
		await page.waitForURL("/", { timeout: 10000 });

		// Verify user is logged in by checking user email in layout (user was activated)
		await expect(page.getByTestId("layout-user-email")).toContainText(TestUsersData.sso_2.email);
	});
});

test.describe("SSO Login - Security Tests", () => {
	test.beforeEach(async () => {
		await resetAndSeedDatabase();
	});

	test("should reject SSO login for unauthorized domain", async ({ page }) => {
		// ssoinvaliduser@differentdomain.test - no tenant exists for this domain
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter email with unauthorized domain
		await page.locator("#email").fill("ssoinvaliduser@differentdomain.test");
		await page.getByTestId("sso-login-submit-button").click();

		// Backend fails at sso_login.go:105 - no tenant found for domain
		// Should show toast error, NOT redirect to Keycloak
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("SSOログインに失敗しました。");

		// Verify we remain on SSO login page (no redirect to Keycloak)
		await expect(page).toHaveURL("/auth/sso/login");
	});

	test("should reject suspended user SSO login", async ({ page }) => {
		// sso3@sso.test is suspended in seed data
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter suspended user email
		await page.locator("#email").fill(TestUsersData.sso_3.email);
		await page.getByTestId("sso-login-submit-button").click();

		// Wait for redirect to Keycloak login page
		await page.waitForURL(/.*mock-keycloak.*/, { timeout: 10000 });

		// Fill in Keycloak credentials
		await page.locator("#username").fill(TestUsersData.sso_3.email);
		await page.locator("#password").fill(TestUsersData.sso_3.plainTextPassword);

		// Submit Keycloak login form
		await page.locator('input[type="submit"][name="login"]').click();

		// Backend validates user status in ACS and should redirect with error
		await page.waitForURL(/\/auth\/sso\/login\?error=/, { timeout: 10000 });

		// Verify error alert is displayed
		await expect(page.getByTestId("sso-login-error-message")).toBeVisible();
		await expect(page.getByTestId("sso-login-error-message")).toContainText(
			"アカウントが停止されています。サポートにお問い合わせください。"
		);
	});

	test("should reject SSO login for password-only tenant", async ({ page }) => {
		// enterprise_1 belongs to a tenant with auth_method="password"
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter email for user whose tenant uses password auth
		await page.locator("#email").fill(TestUsersData.enterprise_1.email);
		await page.getByTestId("sso-login-submit-button").click();

		// Backend fails at sso_login endpoint - tenant has auth_method="password"
		// Should show toast error, NOT redirect to Keycloak
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("SSOログインに失敗しました。");

		// Verify we remain on SSO login page (no redirect to Keycloak)
		await expect(page).toHaveURL("/auth/sso/login");
	});

	test("should reject SSO login when SSO feature is disabled", async ({ page }) => {
		// ssodisabled_1 belongs to a tenant with SSO feature disabled in enterprise_features
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter email for user whose tenant has SSO disabled
		await page.locator("#email").fill(TestUsersData.ssodisabled_1.email);
		await page.getByTestId("sso-login-submit-button").click();

		// Backend fails at sso_login endpoint - SSO feature not enabled
		// Should show toast error, NOT redirect to Keycloak
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("SSOログインに失敗しました。");

		// Verify we remain on SSO login page (no redirect to Keycloak)
		await expect(page).toHaveURL("/auth/sso/login");
	});
});

test.describe("SSO Login - Frontend Validation", () => {
	test.beforeEach(async () => {
		await resetAndSeedDatabase();
	});

	test("should show error when email field is empty", async ({ page }) => {
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Click submit button without entering email
		await page.getByTestId("sso-login-submit-button").click();

		// Should show email validation error
		await expect(page.getByTestId("email-error")).toBeVisible();
		await expect(page.getByTestId("email-error")).toContainText("メールアドレスは必須です");

		// Verify we remain on SSO login page (no API call made)
		await expect(page).toHaveURL("/auth/sso/login");
	});

	test("should show error when email format is invalid", async ({ page }) => {
		// Navigate to SSO login page
		await page.goto("/auth/sso/login");

		// Enter invalid email format
		await page.locator("#email").fill("notanemail");
		await page.getByTestId("sso-login-submit-button").click();

		// Should show email format validation error
		await expect(page.getByTestId("email-error")).toBeVisible();
		await expect(page.getByTestId("email-error")).toContainText(
			"メールアドレスの形式が正しくありません"
		);

		// Verify we remain on SSO login page (no API call made)
		await expect(page).toHaveURL("/auth/sso/login");
	});
});
