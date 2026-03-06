import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "$lugia-test/e2e/setup/helpers";
import { TestUsersData } from "$lugia-test/e2e/setup/seed";
import crypto from "node:crypto";

const CREATE_TENANT_JWT_SECRET = "test_create_tenant_jwt_secret_for_testing_only";

interface TenantSignupTokenPayload {
	email: string;
	company_name: string;
	user_name: string;
	sso?: {
		enabled: boolean;
		idp_metadata_url?: string;
		allowed_domains?: string[];
	};
	iat?: number;
	exp?: number;
}

function createTenantSignupToken(payload: TenantSignupTokenPayload): string {
	const header = { alg: "HS256", typ: "JWT" };
	const now = Math.floor(Date.now() / 1000);

	const fullPayload = {
		...payload,
		iat: payload.iat ?? now,
		exp: payload.exp ?? now + 48 * 60 * 60
	};

	const encodedHeader = Buffer.from(JSON.stringify(header)).toString("base64url");
	const encodedPayload = Buffer.from(JSON.stringify(fullPayload)).toString("base64url");

	const signature = crypto
		.createHmac("sha256", CREATE_TENANT_JWT_SECRET)
		.update(`${encodedHeader}.${encodedPayload}`)
		.digest("base64url");

	return `${encodedHeader}.${encodedPayload}.${signature}`;
}

function createExpiredTenantSignupToken(payload: TenantSignupTokenPayload): string {
	const past = Math.floor(Date.now() / 1000) - 3600;
	return createTenantSignupToken({
		...payload,
		iat: past - 48 * 60 * 60,
		exp: past
	});
}

test.describe("Auth - Tenant Signup Page", () => {
	const tenantSignupURL = "/auth/tenant-signup";
	const loginURL = "/auth/login";

	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.describe("Error state - missing or invalid token", () => {
		test("should show error state when no token is provided", async ({ page }) => {
			await page.goto(tenantSignupURL);

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toContainText("エラー");
			await expect(page.getByTestId("error-message")).toContainText(
				"招待リンクが無効か、期限切れです。"
			);
			await expect(page.getByTestId("login-link")).toBeVisible();

			// Form should not be visible
			await expect(page.getByTestId("signup-form")).not.toBeVisible();
		});

		test("should show error state when token is empty", async ({ page }) => {
			await page.goto(`${tenantSignupURL}?token=`);

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toContainText("エラー");
			await expect(page.getByTestId("signup-form")).not.toBeVisible();
		});

		test("should show error state when token is malformed", async ({ page }) => {
			await page.goto(`${tenantSignupURL}?token=not-a-valid-jwt`);

			// Malformed token can't be decoded by the frontend, so it falls back to error state
			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("signup-form")).not.toBeVisible();
		});

		test("should navigate to login page from error state", async ({ page }) => {
			await page.goto(tenantSignupURL);

			await expect(page.getByTestId("login-link")).toBeVisible();
			await page.getByTestId("login-link").click();
			await expect(page).toHaveURL(loginURL);
		});
	});

	test.describe("Form display with valid token", () => {
		test("should display form with pre-filled data from token", async ({ page }) => {
			const token = createTenantSignupToken({
				email: "newcompany@example.com",
				company_name: "Test Company",
				user_name: "Test User"
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);

			await expect(page.getByTestId("signup-title")).toBeVisible();
			await expect(page.getByTestId("signup-title")).toContainText("アカウントを作成");
			await expect(page.getByTestId("signup-form")).toBeVisible();

			// Email should be pre-filled and disabled
			const emailField = page.locator("#email");
			await expect(emailField).toHaveValue("newcompany@example.com");
			await expect(emailField).toBeDisabled();

			// Company name and user name should be pre-filled
			await expect(page.locator("#company_name")).toHaveValue("Test Company");
			await expect(page.locator("#user_name")).toHaveValue("Test User");

			// Password fields should be visible and empty
			await expect(page.locator("#password")).toBeVisible();
			await expect(page.locator("#password")).toHaveValue("");
			await expect(page.locator("#password_confirm")).toBeVisible();
			await expect(page.locator("#password_confirm")).toHaveValue("");
		});

		test("should hide password fields when SSO is enabled in token", async ({ page }) => {
			const token = createTenantSignupToken({
				email: "ssonew@sso.test",
				company_name: "SSO Company",
				user_name: "SSO User",
				sso: {
					enabled: true,
					idp_metadata_url: "http://mock-keycloak:27001/realms/test-realm/protocol/saml/descriptor",
					allowed_domains: ["sso.test"]
				}
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);

			await expect(page.getByTestId("signup-form")).toBeVisible();
			await expect(page.locator("#email")).toHaveValue("ssonew@sso.test");
			await expect(page.locator("#company_name")).toHaveValue("SSO Company");
			await expect(page.locator("#user_name")).toHaveValue("SSO User");

			// Password fields should not be visible for SSO
			await expect(page.locator("#password")).not.toBeVisible();
			await expect(page.locator("#password_confirm")).not.toBeVisible();
		});
	});

	test.describe("Form validation - password signup", () => {
		let validToken: string;

		test.beforeAll(() => {
			validToken = createTenantSignupToken({
				email: `validation_test_${Date.now()}@example.com`,
				company_name: "",
				user_name: ""
			});
		});

		test("should display validation errors for empty required fields", async ({ page }) => {
			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(validToken)}`);

			await expect(page.getByTestId("signup-form")).toBeVisible();

			// Clear any pre-filled values and submit
			await page.locator("#company_name").fill("");
			await page.locator("#user_name").fill("");
			await page.getByTestId("signup-button").click();

			await expect(page.getByTestId("company_name-error")).toContainText("会社名は必須です", {
				timeout: 7000
			});
			await expect(page.getByTestId("user_name-error")).toContainText("氏名は必須です");
			await expect(page.getByTestId("password-error")).toContainText("パスワードは必須です");
			await expect(page.getByTestId("password_confirm-error")).toContainText(
				"パスワードを確認してください"
			);
		});

		test("should display validation error for password too short", async ({ page }) => {
			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(validToken)}`);

			await page.locator("#password").fill("1234567");
			await page.getByTestId("signup-button").click();

			await expect(page.getByTestId("password-error")).toContainText(
				"パスワードは8文字以上である必要があります",
				{ timeout: 7000 }
			);
		});

		test("should display validation error when passwords do not match", async ({ page }) => {
			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(validToken)}`);

			await page.locator("#password").fill("password123");
			await page.locator("#password_confirm").fill("password456");
			await page.getByTestId("signup-button").click();

			await expect(page.getByTestId("password_confirm-error")).toContainText(
				"パスワードが一致しません",
				{ timeout: 7000 }
			);
		});
	});

	test.describe("Form submission", () => {
		test("should successfully create tenant and redirect to home", async ({ page, baseURL }) => {
			const uniqueEmail = `tenant_signup_${Date.now()}@example.com`;
			const token = createTenantSignupToken({
				email: uniqueEmail,
				company_name: "New Test Company",
				user_name: "New User"
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);
			await expect(page.getByTestId("signup-form")).toBeVisible();

			await page.locator("#password").fill("validPassword123");
			await page.locator("#password_confirm").fill("validPassword123");

			await page.getByTestId("signup-button").click();

			// Wait for the /api/me call that happens after successful signup and auto-login
			await page.waitForResponse(
				(response) => {
					return response.url().includes("/api/me") && response.status() === 200;
				},
				{ timeout: 15000 }
			);

			const expectedHomePageURL = baseURL && baseURL.endsWith("/") ? baseURL : `${baseURL}/`;
			await expect(page).toHaveURL(expectedHomePageURL, { timeout: 15000 });
		});

		test("should show error toast when email already exists", async ({ page }) => {
			const token = createTenantSignupToken({
				email: TestUsersData.enterprise_1.email,
				company_name: "Duplicate Company",
				user_name: "Duplicate User"
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);
			await expect(page.getByTestId("signup-form")).toBeVisible();

			await page.locator("#password").fill("validPassword123");
			await page.locator("#password_confirm").fill("validPassword123");

			await page.getByTestId("signup-button").click();

			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("このメールアドレスは既に使用されています。");
		});

		test("should show error toast when token is expired", async ({ page }) => {
			const uniqueEmail = `expired_tenant_${Date.now()}@example.com`;
			const token = createExpiredTenantSignupToken({
				email: uniqueEmail,
				company_name: "Expired Company",
				user_name: "Expired User"
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);

			// The frontend decodes the JWT payload without verification,
			// so the form still displays with an expired token
			await expect(page.getByTestId("signup-form")).toBeVisible();

			await page.locator("#company_name").fill("Expired Company");
			await page.locator("#user_name").fill("Expired User");
			await page.locator("#password").fill("validPassword123");
			await page.locator("#password_confirm").fill("validPassword123");

			await page.getByTestId("signup-button").click();

			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("無効または期限切れの招待リンクです。");
		});

		test("should show error toast when token has wrong signature", async ({ page }) => {
			// Craft a JWT with a different secret
			const header = { alg: "HS256", typ: "JWT" };
			const now = Math.floor(Date.now() / 1000);
			const payload = {
				email: `wrongsig_${Date.now()}@example.com`,
				company_name: "Forged Company",
				user_name: "Forged User",
				iat: now,
				exp: now + 48 * 60 * 60
			};

			const encodedHeader = Buffer.from(JSON.stringify(header)).toString("base64url");
			const encodedPayload = Buffer.from(JSON.stringify(payload)).toString("base64url");
			const signature = crypto
				.createHmac("sha256", "wrong_secret_key")
				.update(`${encodedHeader}.${encodedPayload}`)
				.digest("base64url");

			const token = `${encodedHeader}.${encodedPayload}.${signature}`;

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);

			// Frontend can still decode the payload, so form displays
			await expect(page.getByTestId("signup-form")).toBeVisible();

			await page.locator("#company_name").fill("Forged Company");
			await page.locator("#user_name").fill("Forged User");
			await page.locator("#password").fill("validPassword123");
			await page.locator("#password_confirm").fill("validPassword123");

			await page.getByTestId("signup-button").click();

			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("無効または期限切れの招待リンクです。");
		});
	});

	test.describe("Form validation - SSO signup", () => {
		test("should validate company name and user name for SSO signup", async ({ page }) => {
			const token = createTenantSignupToken({
				email: `sso_validation_${Date.now()}@sso.test`,
				company_name: "",
				user_name: "",
				sso: {
					enabled: true,
					idp_metadata_url: "http://mock-keycloak:27001/realms/test-realm/protocol/saml/descriptor",
					allowed_domains: ["sso.test"]
				}
			});

			await page.goto(`${tenantSignupURL}?token=${encodeURIComponent(token)}`);
			await expect(page.getByTestId("signup-form")).toBeVisible();

			// Password fields should not be visible
			await expect(page.locator("#password")).not.toBeVisible();
			await expect(page.locator("#password_confirm")).not.toBeVisible();

			// Submit with empty fields
			await page.locator("#company_name").fill("");
			await page.locator("#user_name").fill("");
			await page.getByTestId("signup-button").click();

			await expect(page.getByTestId("company_name-error")).toContainText("会社名は必須です", {
				timeout: 7000
			});
			await expect(page.getByTestId("user_name-error")).toContainText("氏名は必須です");
		});
	});
});
