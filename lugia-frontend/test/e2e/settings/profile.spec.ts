import { test, expect, APIRequestContext } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";
import { logInAs } from "../setup/auth";

const PROFILE_URL = "/settings/profile";
const MOCK_SENDGRID_API_URL = "http://mock-sendgrid:27000/json";
const MOCK_SENDGRID_API_KEY = "e2e_fake_sendgrid_key";

// Helper function to get the email change token from mock-sendgrid
async function getEmailChangeToken(
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
			const tokenMatch = htmlBody.match(/verify\/change-email\?token=([a-zA-Z0-9-_.]+)/);
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
	console.error(`Could not find email change token for ${email} after ${retries} retries.`);
	return null;
}

test.describe("Settings - Profile Page", () => {
	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.describe("Authentication and Access Control", () => {
		test("should redirect to login page when not authenticated", async ({ page }) => {
			await page.goto(PROFILE_URL);
			await expect(page).toHaveURL(/.*\/auth\/login.*/);
		});

		test("should allow authenticated users access to profile page", async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_editor);
			await page.goto(PROFILE_URL);

			await expect(page).toHaveURL(PROFILE_URL);
			await expect(page.getByTestId("page-title")).toContainText("プロフィール設定");
		});

		test("should show tenant section only for admin users", async ({ page }) => {
			// Test as admin - should see tenant section
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(PROFILE_URL);

			await expect(page.getByTestId("change-tenant-section")).toBeVisible();
			await expect(page.getByTestId("change-tenant-heading")).toBeVisible();
		});

		test("should hide tenant section for editor users", async ({ page }) => {
			// Test as editor - should not see tenant section
			await logInAs(page, TestUsersData.alpha_editor);
			await page.goto(PROFILE_URL);

			await expect(page.getByTestId("change-tenant-section")).not.toBeVisible();
			await expect(page.getByTestId("change-tenant-heading")).not.toBeVisible();
		});
	});

	test.describe("Name Change Form", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_editor);
			await page.goto(PROFILE_URL);
			await expect(page.getByTestId("change-name-section")).toBeVisible();
		});

		test("should display validation error for empty name", async ({ page }) => {
			// Clear the name field and submit
			await page.locator("#name").clear();
			await page.getByTestId("save-name-button").click();

			await expect(page.getByTestId("name-error")).toContainText("氏名は必須です");
		});

		test("should successfully change user name", async ({ page }) => {
			const newName = "Updated Editor Name";

			// Fill new name and submit
			await page.locator("#name").fill(newName);
			await page.getByTestId("save-name-button").click();

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("氏名を更新しました。");

			// Verify the form shows the updated name
			await expect(page.locator("#name")).toHaveValue(newName);

			// Verify persistence by navigating away and back
			await page.goto("/");
			await page.goto(PROFILE_URL);
			await expect(page.locator("#name")).toHaveValue(newName);
		});
	});

	test.describe("Password Change Form", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_editor);
			await page.goto(PROFILE_URL);
			await expect(page.getByTestId("change-password-section")).toBeVisible();
		});

		test("should display validation errors for empty fields", async ({ page }) => {
			await page.getByTestId("save-password-button").click();

			await expect(page.getByTestId("currentPassword-error")).toContainText(
				"現在のパスワードは必須です"
			);
			await expect(page.getByTestId("newPassword-error")).toContainText(
				"新しいパスワードは必須です"
			);
			await expect(page.getByTestId("confirmPassword-error")).toContainText(
				"新しいパスワード（確認）は必須です"
			);
		});

		test("should display error for password mismatch", async ({ page }) => {
			await page.locator("#currentPassword").fill("password123");
			await page.locator("#newPassword").fill("newPassword123");
			await page.locator("#confirmPassword").fill("differentPassword123");
			await page.getByTestId("save-password-button").click();

			await expect(page.getByTestId("confirmPassword-error")).toContainText(
				"パスワードが一致しません"
			);
		});

		test("should display error for password too short", async ({ page }) => {
			await page.locator("#currentPassword").fill("password123");
			await page.locator("#newPassword").fill("short");
			await page.locator("#confirmPassword").fill("short");
			await page.getByTestId("save-password-button").click();

			await expect(page.getByTestId("newPassword-error")).toContainText(
				"パスワードは8文字以上である必要があります"
			);
		});

		test("should successfully change password and update authentication", async ({ page }) => {
			const currentPassword = TestUsersData.alpha_editor.plainTextPassword;
			const newPassword = "newPassword123";

			// Fill password change form
			await page.locator("#currentPassword").fill(currentPassword);
			await page.locator("#newPassword").fill(newPassword);
			await page.locator("#confirmPassword").fill(newPassword);
			await page.getByTestId("save-password-button").click();

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("パスワードを更新しました。");

			// Verify form is reset (all fields should be empty)
			await expect(page.locator("#currentPassword")).toHaveValue("");
			await expect(page.locator("#newPassword")).toHaveValue("");
			await expect(page.locator("#confirmPassword")).toHaveValue("");

			// Log out to test password change effectiveness
			await page.getByTestId("navigation-signout").click();
			await expect(page).toHaveURL(/.*\/auth\/login.*/);

			// Try to log in with old password - should fail
			await page.locator("#email").fill(TestUsersData.alpha_editor.email);
			await page.locator("#password").fill(currentPassword);
			await page.getByTestId("login-submit-button").click();

			// Should show error for old password
			const errorToast = page.getByTestId("toast-0");
			await expect(errorToast).toBeVisible({ timeout: 10000 });
			await expect(errorToast).toContainText("メールアドレスまたはパスワードが正しくありません");

			// Clear form and try with new password - should succeed
			await page.locator("#password").clear();
			await page.locator("#password").fill(newPassword);
			await page.getByTestId("login-submit-button").click();

			// Should be redirected to dashboard
			await expect(page).toHaveURL("/", { timeout: 10000 });
		});
	});

	test.describe("Email Change Form", () => {
		test.beforeEach(async ({ page }) => {
			// Use beta_admin instead of alpha_editor to avoid rate limiting conflicts
			await logInAs(page, TestUsersData.beta_admin);
			await page.goto(PROFILE_URL);
			await expect(page.getByTestId("change-email-section")).toBeVisible();
		});

		test("should display current email address", async ({ page }) => {
			await expect(page.getByTestId("current-email")).toContainText(TestUsersData.beta_admin.email);
		});

		test("should display validation errors for email field", async ({ page }) => {
			// Test empty email
			await page.getByTestId("save-email-button").click();
			await expect(page.getByTestId("newEmail-error")).toContainText(
				"新しいメールアドレスは必須です"
			);

			// Test invalid email format
			await page.locator("#newEmail").fill("invalid-email");
			await page.getByTestId("save-email-button").click();
			await expect(page.getByTestId("newEmail-error")).toContainText(
				"有効なメールアドレスを入力してください"
			);

			// Test same email as current
			await page.locator("#newEmail").fill(TestUsersData.beta_admin.email);
			await page.getByTestId("save-email-button").click();
			await expect(page.getByTestId("newEmail-error")).toContainText(
				"現在のメールアドレスと同じです"
			);
		});

		test("should successfully request email change and send verification email", async ({
			page,
			request
		}) => {
			const newEmail = "beta_admin_new@example.com";

			// Fill new email and submit
			await page.locator("#newEmail").fill(newEmail);
			await page.getByTestId("save-email-button").click();

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText(
				"確認メールを送信しました。メールをご確認ください。"
			);

			// Verify form is reset
			await expect(page.locator("#newEmail")).toHaveValue("");

			// Verify email was sent by checking mock SendGrid
			const token = await getEmailChangeToken(request, newEmail);
			expect(token).not.toBeNull();
		});
	});

	test.describe("Tenant Name Change (Admin Only)", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(PROFILE_URL);
			await expect(page.getByTestId("change-tenant-section")).toBeVisible();
		});

		test("should display validation error for empty tenant name", async ({ page }) => {
			// Clear the tenant name field and submit
			await page.locator("#tenantName").clear();
			await page.getByTestId("save-tenant-button").click();

			await expect(page.getByTestId("tenantName-error")).toContainText("組織名は必須です");
		});

		test("should successfully change tenant name", async ({ page }) => {
			const newTenantName = "Updated Tenant Alpha";

			// Fill new tenant name and submit
			await page.locator("#tenantName").fill(newTenantName);
			await page.getByTestId("save-tenant-button").click();

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("組織名を更新しました。");

			// Verify the form shows the updated tenant name
			await expect(page.locator("#tenantName")).toHaveValue(newTenantName);

			// Verify persistence by navigating away and back
			await page.goto("/");
			await page.goto(PROFILE_URL);
			await expect(page.locator("#tenantName")).toHaveValue(newTenantName);
		});
	});

	test.describe("Email Verification Success Handling", () => {
		test("should show success toast when returning from email verification", async ({ page }) => {
			// Use alpha_admin to avoid rate limiting issues with alpha_editor
			await logInAs(page, TestUsersData.alpha_admin);

			// Navigate to profile page with email-verified=true parameter
			await page.goto(`${PROFILE_URL}?email-verified=true`);

			// Should show success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("メールアドレスの変更が完了しました。");
		});

		test("should not show toast without email-verified parameter", async ({ page }) => {
			// Use alpha_admin to avoid rate limiting issues with alpha_editor
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(PROFILE_URL);

			// Should not show any toast
			await expect(page.getByTestId("toast-0")).not.toBeVisible();
		});
	});
});
