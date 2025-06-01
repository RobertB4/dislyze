import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";

// Use valid tokens from seed data
const VALID_TOKEN = "26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ";
const INVALID_TOKEN = "invalid_token_12345";
const INVITER_NAME = "Test Inviter";
const INVITED_EMAIL = "pending_editor_valid_token@example.com";

test.beforeAll(async () => {
	await resetAndSeedDatabase();
});

test.describe("Accept Invite Page", () => {
	test.describe("Missing Parameters", () => {
		test("should show error when no parameters provided", async ({ page }) => {
			await page.goto("/auth/accept-invite");

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("login-link")).toBeVisible();
			await expect(page.getByTestId("accept-invite-form")).not.toBeVisible();
		});

		test("should show error when only token is provided", async ({ page }) => {
			await page.goto(`/auth/accept-invite?token=${VALID_TOKEN}`);

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("accept-invite-form")).not.toBeVisible();
		});

		test("should show error when token is missing", async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("accept-invite-form")).not.toBeVisible();
		});
	});

	test.describe("Valid Parameters - Form Display", () => {
		test("should show form with valid parameters", async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?token=${VALID_TOKEN}&inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);

			await expect(page.getByTestId("accept-invite-title")).toBeVisible();
			await expect(page.getByTestId("inviter-message")).toBeVisible();

			await expect(page.getByTestId("accept-invite-form")).toBeVisible();
			await expect(page.locator("#email")).toHaveValue(INVITED_EMAIL);
			await expect(page.locator("#email")).toBeDisabled();
			await expect(page.locator("#password")).toBeVisible();
			await expect(page.locator("#password_confirm")).toBeVisible();
			await expect(page.getByTestId("submit-button")).toBeVisible();
		});

		test("should show error when inviter name is not provided", async ({ page }) => {
			await page.goto(`/auth/accept-invite?token=${VALID_TOKEN}&invited_email=${INVITED_EMAIL}`);

			await expect(page.getByTestId("error-state")).toBeVisible();
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("accept-invite-form")).not.toBeVisible();
		});
	});

	test.describe("Form Validation", () => {
		test.beforeEach(async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?token=${VALID_TOKEN}&inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);
		});

		test("should show password required error", async ({ page }) => {
			// Leave both password fields empty and try to trigger validation
			const passwordInput = page.locator("#password");

			// Focus and blur password field to trigger validation
			await passwordInput.focus();
			await passwordInput.blur();

			const passwordError = page.getByTestId("password-error");
			await expect(passwordError).toBeVisible();
			await expect(passwordError).toContainText("パスワードは必須です");
			await expect(page.getByTestId("submit-button")).toBeDisabled();
		});

		test("should show password length error for short password", async ({ page }) => {
			const passwordInput = page.locator("#password");
			await passwordInput.fill("123");
			await passwordInput.blur();

			const passwordError = page.getByTestId("password-error");
			await expect(passwordError).toBeVisible();
			await expect(passwordError).toContainText("パスワードは8文字以上である必要があります");
			await expect(page.getByTestId("submit-button")).toBeDisabled();
		});

		test("should show password confirmation required error", async ({ page }) => {
			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");

			// Fill password but leave confirmation empty
			await passwordInput.fill("password123");
			await passwordConfirmInput.focus();
			await passwordConfirmInput.blur();

			const passwordConfirmError = page.getByTestId("password_confirm-error");
			await expect(passwordConfirmError).toBeVisible();
			await expect(passwordConfirmError).toContainText("パスワードを確認してください");
			await expect(page.getByTestId("submit-button")).toBeDisabled();
		});

		test("should show password mismatch error", async ({ page }) => {
			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");

			await passwordInput.fill("password123");
			await passwordConfirmInput.fill("different123");
			await passwordConfirmInput.blur();

			const passwordConfirmError = page.getByTestId("password_confirm-error");
			await expect(passwordConfirmError).toBeVisible();
			await expect(passwordConfirmError).toContainText("パスワードが一致しません");
			await expect(page.getByTestId("submit-button")).toBeDisabled();
		});

		test("should enable submit button with valid passwords", async ({ page }) => {
			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");

			await passwordInput.fill("password123");
			await passwordConfirmInput.fill("password123");
			await passwordConfirmInput.blur();

			await expect(page.getByTestId("submit-button")).toBeEnabled();
		});

		test("should trim whitespace from passwords", async ({ page }) => {
			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");

			await passwordInput.fill("  password123  ");
			await passwordConfirmInput.fill("  password123  ");
			await passwordConfirmInput.blur();

			await expect(page.getByTestId("submit-button")).toBeEnabled();
		});
	});

	test.describe("Form Submission", () => {
		test("should successfully accept invitation with valid token", async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?token=${VALID_TOKEN}&inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);

			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");
			const submitButton = page.getByTestId("submit-button");

			await passwordInput.fill("password123");
			await passwordConfirmInput.fill("password123");
			await submitButton.click();

			// Wait for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("招待が承認されました。");

			// Should redirect to home page
			await page.waitForURL("/");
		});

		test("should handle API error - invalid token", async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?token=${INVALID_TOKEN}&inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);

			const passwordInput = page.locator("#password");
			const passwordConfirmInput = page.locator("#password_confirm");
			const submitButton = page.getByTestId("submit-button");

			await passwordInput.fill("password123");
			await passwordConfirmInput.fill("password123");
			await submitButton.click();

			// Should show error toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText(
				"招待リンクが無効か、期限切れです。お手数ですが、招待者に再度依頼してください。"
			);
		});
	});

	test.describe("Security", () => {
		test("should not expose sensitive data in form", async ({ page }) => {
			await page.goto(
				`/auth/accept-invite?token=${VALID_TOKEN}&inviter_name=${INVITER_NAME}&invited_email=${INVITED_EMAIL}`
			);

			// Token should not be visible in the page content
			await expect(page.getByText(VALID_TOKEN)).not.toBeVisible();

			// Email should be disabled (read-only)
			await expect(page.locator("#email")).toBeDisabled();
		});

		test("should handle malicious parameters safely", async ({ page }) => {
			const maliciousScript = "<script>alert('xss')</script>";
			const maliciousName = `Evil${maliciousScript}User`;
			const maliciousEmail = `evil${maliciousScript}@example.com`;

			await page.goto(
				`/auth/accept-invite?token=${VALID_TOKEN}&inviter_name=${encodeURIComponent(maliciousName)}&invited_email=${encodeURIComponent(maliciousEmail)}`
			);

			// Should display safely escaped content
			await expect(page.getByTestId("accept-invite-form")).toBeVisible();
			await expect(page.locator("#email")).toHaveValue(maliciousEmail);

			// Script should not execute
			page.on("dialog", () => {
				throw new Error("XSS script executed");
			});
		});
	});
});
