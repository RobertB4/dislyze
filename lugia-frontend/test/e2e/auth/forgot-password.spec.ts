import { test, expect, APIRequestContext } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";

const FORGOT_PASSWORD_URL = "/auth/forgot-password";
const RESET_PASSWORD_URL = "/auth/reset-password";
const LOGIN_URL = "/auth/login";
const MOCK_SENDGRID_API_URL = "http://mock-sendgrid:27000/json";
const MOCK_SENDGRID_API_KEY = "e2e_fake_sendgrid_key";

// Helper function to get the reset token from mock-sendgrid
async function getResetToken(
	request: APIRequestContext,
	email: string,
	retries = 5,
	delay = 2000
): Promise<string | null> {
	for (let i = 0; i < retries; i++) {
		try {
			const response = await request.get(`${MOCK_SENDGRID_API_URL}?token=${MOCK_SENDGRID_API_KEY}`);
			if (!response.ok()) {
				console.error(`Mock SendGrid API request failed with status: ${response.status()}`);
				await new Promise((resolve) => setTimeout(resolve, delay));
				continue;
			}
			const allEmails = (await response.json()) as any[];

			if (allEmails.length === 0) {
				throw new Error("No emails found in mock-sendgrid response");
			}
			const targetEmail = allEmails[0];
			const htmlContent = targetEmail.content?.find((c: any) => c.type === "text/html");
			if (!htmlContent || !htmlContent.value) {
				throw new Error("No HTML content found in the email.");
			}
			const htmlBody = htmlContent.value;
			const tokenMatch = htmlBody.match(/reset-password\?token=([a-zA-Z0-9-_.]+)/);
			if (tokenMatch && tokenMatch[1]) {
				return tokenMatch[1];
			}
		} catch (error) {
			console.error("Error fetching or parsing emails from mock-sendgrid:", error);
		}
		if (i < retries - 1) {
			console.log(`Retrying to fetch email for ${email}, attempt ${i + 1}/${retries}`);
			await new Promise((resolve) => setTimeout(resolve, delay));
		}
	}
	console.error(`Could not find reset token for ${email} after ${retries} retries.`);
	return null;
}

test.describe("Auth - Forgot Password & Reset Password", () => {
	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.describe("Forgot Password Page", () => {
		test.beforeEach(async ({ page }) => {
			await page.goto(FORGOT_PASSWORD_URL);
		});

		test("should load the page correctly", async ({ page }) => {
			await expect(page.getByTestId("forgot-password-heading")).toBeVisible();
			await expect(page.locator("#email")).toBeVisible();
			await expect(page.getByTestId("forgot-password-submit-button")).toBeVisible();
			await expect(page.getByTestId("back-to-login-link")).toBeVisible();
		});

		test("should display validation error for empty email", async ({ page }) => {
			await page.getByTestId("forgot-password-submit-button").click();
			await expect(page.getByTestId("email-error")).toContainText("メールアドレスは必須です", {
				timeout: 7000
			});
		});

		test("should display validation error for invalid email format", async ({ page }) => {
			await page.locator("#email").fill("invalidemail");
			await page.getByTestId("forgot-password-submit-button").click();
			await expect(page.getByTestId("email-error")).toContainText(
				"メールアドレスの形式が正しくありません",
				{ timeout: 7000 }
			);
		});

		test("should show success toast for non-existent email (to prevent user enumeration)", async ({
			page
		}) => {
			await page.locator("#email").fill("nonexistent@example.com");
			await page.getByTestId("forgot-password-submit-button").click();
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText(
				"パスワードリセットの手順を記載したメールを送信しました。"
			);
			await expect(page).toHaveURL(FORGOT_PASSWORD_URL); // Should stay on the page
		});

		test("should show success toast and send email for existing user", async ({
			page,
			request
		}) => {
			await page.locator("#email").fill(TestUsersData.alpha_editor.email);
			await page.getByTestId("forgot-password-submit-button").click();

			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText(
				"パスワードリセットの手順を記載したメールを送信しました。"
			);
			await expect(page).toHaveURL(FORGOT_PASSWORD_URL);

			const token = await getResetToken(request, TestUsersData.alpha_editor.email);
			expect(token).not.toBeNull();
		});
	});

	test.describe("Reset Password Page - Direct Access & Invalid Token", () => {
		test("should show SvelteKit error page if no token is provided", async ({ page }) => {
			await page.goto(RESET_PASSWORD_URL);
			// Check for SvelteKit +error.svelte page content
			await expect(page.getByTestId("error-title")).toContainText("エラーが発生しました (400)", {
				timeout: 10000
			});
			await expect(page.getByTestId("error-message")).toContainText(
				"このパスワードリセットリンクは無効か期限切れです。お手数ですが、再度リセットをリクエストしてください。",
				{ timeout: 7000 }
			);
			// Ensure elements from the actual reset-password page are not visible
			await expect(page.getByTestId("reset-password-heading")).not.toBeVisible();
			await expect(page.getByTestId("toast-0")).not.toBeVisible(); // No toast should be shown
		});

		test("should show SvelteKit error page if token is invalid", async ({ page }) => {
			await page.goto(`${RESET_PASSWORD_URL}?token=invalidtoken123`);
			// Check for SvelteKit +error.svelte page content
			await expect(page.getByTestId("error-title")).toContainText("エラーが発生しました (400)", {
				timeout: 10000
			});
			await expect(page.getByTestId("error-message")).toContainText(
				"このパスワードリセットリンクは無効か期限切れです。お手数ですが、再度リセットをリクエストしてください。",
				{ timeout: 7000 }
			);
			// Ensure elements from the actual reset-password page are not visible
			await expect(page.getByTestId("reset-password-heading")).not.toBeVisible();
			await expect(page.getByTestId("reset-password-token-error-message")).not.toBeVisible();
			await expect(page.locator("#password")).not.toBeVisible();
			await expect(page.locator("#password_confirm")).not.toBeVisible();
			await expect(page.getByTestId("reset-password-submit-button")).not.toBeVisible();
		});
	});

	test.describe("End-to-End Password Reset Flow", () => {
		const testUser = TestUsersData.alpha_editor;
		const newPassword = "newPassword";

		test("should allow user to reset password and login with new password", async ({
			page,
			request
		}) => {
			// 1. Request password reset
			await page.goto(FORGOT_PASSWORD_URL);
			await page.locator("#email").fill(testUser.email);
			await page.getByTestId("forgot-password-submit-button").click();
			await expect(page.getByTestId("toast-0")).toBeVisible({ timeout: 10000 });

			// 2. Get reset token from mock SendGrid
			const resetToken = await getResetToken(request, testUser.email);
			expect(resetToken).not.toBeNull();
			if (!resetToken) throw new Error("Reset token not found");

			// 3. Navigate to reset password page with token
			await page.goto(`${RESET_PASSWORD_URL}?token=${resetToken}`);
			await expect(page.getByTestId("reset-password-heading")).toBeVisible();
			await expect(page.locator("#password")).toBeVisible();
			await expect(page.locator("#password_confirm")).toBeVisible();
			await expect(page.getByTestId("reset-password-submit-button")).toBeVisible();

			// 4. Submit new password
			await page.locator("#password").fill(newPassword);
			await page.locator("#password_confirm").fill(newPassword);
			await page.getByTestId("reset-password-submit-button").click();

			// 5. Verify navigation to login
			await expect(page).toHaveURL(LOGIN_URL, { timeout: 10000 });

			// 6. Attempt login with old password (should fail)
			await page.locator("#email").fill(testUser.email);
			await page.locator("#password").fill(testUser.plainTextPassword);
			await page.getByTestId("login-submit-button").click();
			const errorToast = page.getByTestId("toast-0");
			await expect(errorToast).toBeVisible({ timeout: 10000 });
			await expect(errorToast).toContainText("メールアドレスまたはパスワードが正しくありません");
			await expect(page).toHaveURL(LOGIN_URL);

			// 7. Attempt login with new password (should succeed)
			await page.locator("#email").fill(testUser.email);
			await page.locator("#password").fill(newPassword);
			await page.getByTestId("login-submit-button").click();
			await page.waitForResponse(
				(response) => response.url().includes("/api/me") && response.status() === 200,
				{ timeout: 15000 }
			);
			await expect(page).toHaveURL("/", { timeout: 15000 });
		});
	});

	test.describe("Reset Password Page - Form Validation", () => {
		let validResetToken: string | null;

		test.beforeEach(async ({ page, request }) => {
			await resetAndSeedDatabase();

			await page.goto(FORGOT_PASSWORD_URL);
			await page.locator("#email").fill(TestUsersData.alpha_editor.email);
			await page.getByTestId("forgot-password-submit-button").click();
			await expect(page.getByTestId("toast-0")).toBeVisible({ timeout: 10000 });

			validResetToken = await getResetToken(request, TestUsersData.alpha_editor.email);
			expect(
				validResetToken,
				"Failed to obtain a valid reset token for validation tests"
			).not.toBeNull();

			if (validResetToken) {
				await page.goto(`${RESET_PASSWORD_URL}?token=${validResetToken}`);
				await expect(page.getByTestId("reset-password-heading")).toBeVisible();
			} else {
				throw new Error("Failed to get reset token in beforeEach for form validation tests");
			}
		});

		test("should display validation errors for empty fields", async ({ page }) => {
			await page.getByTestId("reset-password-submit-button").click();
			await expect(page.getByTestId("password-error")).toContainText("パスワードは必須です", {
				timeout: 7000
			});
			await expect(page.getByTestId("password_confirm-error")).toContainText(
				"パスワード確認は必須です",
				{ timeout: 7000 }
			);
		});

		test("should display error for mismatched passwords", async ({ page }) => {
			await page.locator("#password").fill("newPassword123!");
			await page.locator("#password_confirm").fill("differentPassword123!");
			await page.getByTestId("reset-password-submit-button").click();
			await expect(page.getByTestId("password_confirm-error")).toContainText(
				"パスワードが一致しません",
				{ timeout: 7000 }
			);
		});

		test("should display error for password too short", async ({ page }) => {
			const shortPassword = "short";
			await page.locator("#password").fill(shortPassword);
			await page.locator("#password_confirm").fill(shortPassword);
			await page.getByTestId("reset-password-submit-button").click();
			await expect(page.getByTestId("password-error")).toContainText(
				"パスワードは8文字以上である必要があります",
				{ timeout: 7000 }
			);
		});
	});
});
