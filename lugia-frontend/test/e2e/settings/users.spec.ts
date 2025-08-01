import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";
import { logInAs } from "../setup/auth";

test.describe("Settings - Users Page", () => {
	const usersPageURL = "/settings/users";

	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	// Authentication and access control tests
	test.describe("Authentication and Access Control", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test("should redirect to login page when not authenticated", async ({ page }) => {
			await page.goto(usersPageURL);
			await expect(page).toHaveURL(/.*\/auth\/login.*/);
		});

		test("should allow admin access to users page", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(usersPageURL);

			await expect(page).toHaveURL(usersPageURL);
			await expect(page.getByRole("heading", { name: "ユーザー管理" })).toBeVisible();
		});

		test("should display 403 error when editor tries to access users page", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_2);
			await page.goto(usersPageURL);

			// Should show the error elements with the correct content
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-title")).toContainText("エラーが発生しました (403)");
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("error-message")).toContainText("権限がありません。");
		});
	});
	// User viewing and search tests
	test.describe("Viewing Users and Search", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block using the helper function
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded - users table should be visible
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should display a list of users", async ({ page }) => {
			// By default, we should see the first 2 users (ordered by created_at DESC)
			// This would be users b...6 and b...5 based on seed data

			// Check for the PendingXT Editor (first user in order)
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_11.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-name-${TestUsersData.enterprise_11.userID}`)
			).toContainText("吉田 雄二");

			// Check for the Rate Limit Test user (second user in order)
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_12.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-name-${TestUsersData.enterprise_12.userID}`)
			).toContainText("福田 あかり");

			// The first page should only contain 2 users due to pagination
			expect(await page.getByTestId(/^user-row-/).count()).toBe(50);
		});

		test("should search for users by name", async ({ page }) => {
			// Enter search term in the search input using id selector
			await page.locator("#user-search").fill("佐藤 花子");

			// Since we're changing the URL with search, wait for navigation to complete
			await page.waitForResponse(
				(response) => response.url().includes("/api/users") && response.status() === 200
			);

			// Verify only matching users are displayed
			await expect(page.getByTestId(`user-row-${TestUsersData.enterprise_2.userID}`)).toBeVisible();

			// User should see their search term in the URL
			await expect(page).toHaveURL(/.*search=%E4%BD%90%E8%97%A4.*%E8%8A%B1%E5%AD%90.*/);

			// Only the editor user should be visible
			expect(await page.getByTestId(/^user-row-/).count()).toBe(1);
		});

		test("should search for users by email", async ({ page }) => {
			// Enter search term in the search input using id selector
			await page.locator("#user-search").fill("enterprise2@localhost.com");

			// Since we're changing the URL with search, wait for navigation to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes("enterprise2") &&
					response.status() === 200
			);

			// Verify only matching users are displayed
			await expect(page.getByTestId(`user-row-${TestUsersData.enterprise_2.userID}`)).toBeVisible();

			// User should see their search term in the URL
			await expect(page).toHaveURL(/.*search=enterprise2%40localhost.com.*/);

			// Only the editor user should be visible
			expect(await page.getByTestId(/^user-row-/).count()).toBe(1);
		});

		test("should show 'no results' message when search has no matches", async ({ page }) => {
			// Enter a search term that shouldn't match any users using id selector
			await page.locator("#user-search").fill("non-existent-user-xyz");

			// Since we're changing the URL with search, wait for navigation to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes("non-existent-user-xyz") &&
					response.status() === 200
			);

			// Verify no results message
			await expect(page.getByTestId("no-users-message")).toBeVisible();
			await expect(page.getByTestId("no-search-results-message")).toContainText(
				"「non-existent-user-xyz」に一致するユーザーはありません"
			);

			// No user rows should be visible
			expect(await page.getByTestId(/^user-row-/).count()).toBe(0);
		});

		test("should display different status badges with appropriate colors", async ({ page }) => {
			// First we'll check the pending status badge on the first page
			const pendingUser = TestUsersData.enterprise_12;
			await expect(page.getByTestId(`user-status-${pendingUser.userID}`)).toBeVisible();

			// For pending users, check the badge color and text
			await expect(page.getByTestId(`user-status-${pendingUser.userID}`)).toContainText("招待済み");
			await expect(page.getByTestId(`user-status-badge-${pendingUser.userID}`)).toHaveClass(
				/bg-yellow-100/
			);

			// Now search for the admin user specifically to test active status
			await page.locator("#user-search").fill(TestUsersData.enterprise_1.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_1.email)) &&
					response.status() === 200
			);

			// Check active user status
			await expect(
				page.getByTestId(`user-status-${TestUsersData.enterprise_1.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-status-${TestUsersData.enterprise_1.userID}`)
			).toContainText("有効");
			await expect(
				page.getByTestId(`user-status-badge-${TestUsersData.enterprise_1.userID}`)
			).toHaveClass(/bg-green-100/);

			// Clear search and search for suspended user
			await page.locator("#user-search").fill(TestUsersData.enterprise_16.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_16.email)) &&
					response.status() === 200
			);

			// Check suspended user status
			await expect(
				page.getByTestId(`user-status-${TestUsersData.enterprise_16.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-status-${TestUsersData.enterprise_16.userID}`)
			).toContainText("停止中");
			await expect(
				page.getByTestId(`user-status-badge-${TestUsersData.enterprise_16.userID}`)
			).toHaveClass(/bg-red-100/);
		});
	});
	// Pagination tests
	test.describe("Pagination", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should display pagination controls for multiple pages", async ({ page }) => {
			// Check that pagination controls are visible
			await expect(page.getByTestId("pagination-controls")).toBeVisible();
			await expect(page.getByTestId("pagination-info")).toBeVisible();
			await expect(page.getByTestId("pagination-buttons")).toBeVisible();

			// Check pagination info shows correct counts (101 total users, showing 1-50)
			await expect(page.getByTestId("pagination-info")).toContainText("101件中 1 - 50件を表示");

			// Check current page indicator
			await expect(page.getByTestId("pagination-current")).toContainText("1 / 3");

			// First page - Previous and First buttons should be disabled
			await expect(page.getByTestId("pagination-first")).toBeDisabled();
			await expect(page.getByTestId("pagination-prev")).toBeDisabled();

			// Next and Last buttons should be enabled
			await expect(page.getByTestId("pagination-next")).not.toBeDisabled();
			await expect(page.getByTestId("pagination-last")).not.toBeDisabled();
		});

		test("should navigate to next page and show correct users", async ({ page }) => {
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_11.userID}`)
			).toBeVisible();

			const responsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=2") &&
					response.status() === 200
				);
			});

			// Click next page button
			await page.getByTestId("pagination-next").click();

			// Wait for API response and page to update
			await responsePromise;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Should be on page 2
			await expect(page.getByTestId("pagination-current")).toContainText("2 / 3");
			await expect(page.getByTestId("pagination-info")).toContainText("101件中 51 - 100件を表示");

			await expect(page.getByTestId(`user-row-a0000000-0000-0000-0000-000000000064`)).toBeVisible();
			// Users from page 1 should not be visible anymore
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_11.userID}`)
			).not.toBeVisible();

			// First and Previous buttons should be enabled now
			await expect(page.getByTestId("pagination-first")).not.toBeDisabled();
			await expect(page.getByTestId("pagination-prev")).not.toBeDisabled();
		});

		test("should navigate to previous page", async ({ page }) => {
			const responsePromise2 = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=2") &&
					response.status() === 200
				);
			});

			// Start by going to page 2
			await page.getByTestId("pagination-next").click();

			// Wait for page 2 response
			await responsePromise2;

			await expect(page.getByTestId("users-table")).toBeVisible();

			const responsePromise1 = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=1") &&
					response.status() === 200
				);
			});

			// Now navigate back to page 1
			await page.getByTestId("pagination-prev").click();

			// Wait for API response and page to update
			await responsePromise1;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Should be back on page 1
			await expect(page.getByTestId("pagination-current")).toContainText("1 / 3");
			await expect(page.getByTestId("pagination-info")).toContainText("101件中 1 - 50件を表示");

			// Page 1 users should be visible again
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_11.userID}`)
			).toBeVisible();
		});

		test("should navigate to last page", async ({ page }) => {
			const responsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=3") &&
					response.status() === 200
				);
			});

			// Go directly to last page
			await page.getByTestId("pagination-last").click();

			// Wait for API response and page to update
			await responsePromise;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Should be on page 14
			await expect(page.getByTestId("pagination-current")).toContainText("3 / 3");
			await expect(page.getByTestId("pagination-info")).toContainText("101件中 101 - 101件を表示");

			await expect(page.getByTestId(`user-row-a0000000-0000-0000-0000-000000000101`)).toBeVisible();

			// Next and Last buttons should be disabled on the last page
			await expect(page.getByTestId("pagination-next")).toBeDisabled();
			await expect(page.getByTestId("pagination-last")).toBeDisabled();
		});

		test("should navigate to first page", async ({ page }) => {
			const responsePromise3 = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=3") &&
					response.status() === 200
				);
			});

			// First go to page 3
			await page.getByTestId("pagination-last").click();

			// Wait for page 3 response
			await responsePromise3;

			await expect(page.getByTestId("users-table")).toBeVisible();

			const responsePromise1 = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("page=1") &&
					response.status() === 200
				);
			});

			// Now click the "first page" button
			await page.getByTestId("pagination-first").click();

			// Wait for API response and page to update
			await responsePromise1;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Should be back on page 1
			await expect(page.getByTestId("pagination-current")).toContainText("1 / 3");

			// First page users should be visible
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_11.userID}`)
			).toBeVisible();
		});

		test("should persist search results when navigating through pages", async ({ page }) => {
			const searchResponsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("search=enterprise") &&
					response.status() === 200
				);
			});

			// Search for "中島" which should match a user
			await page.locator("#user-search").fill("enterprise");

			// Wait for search to complete
			await searchResponsePromise;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Check that pagination reflects the search results
			await expect(page.getByTestId("pagination-info")).toBeVisible();

			const pageResponsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("search=enterprise") &&
					response.url().includes("page=2") &&
					response.status() === 200
				);
			});

			// Navigate to the second page of search results
			await page.getByTestId("pagination-next").click();

			// Wait for API response and page to update
			await pageResponsePromise;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// URL should contain both search term and page number (in any order)
			expect(page.url()).toContain("search=enterprise");
			expect(page.url()).toContain("page=2");

			// Should still be showing search results, but for page 2
			expect(await page.getByTestId(/^user-row-/).count()).toBeGreaterThan(0);
		});
	});
	// User Invitation Tests
	test.describe("User Invitation", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should open invitation form when add user button is clicked", async ({ page }) => {
			// Click the add user button
			await page.getByTestId("add-user-button").click();

			// Verify slideover is displayed
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Verify form fields are present
			await expect(page.locator("#email")).toBeVisible();
			await expect(page.locator("#name")).toBeVisible();

			// Verify role cards are present instead of dropdown
			await expect(page.getByTestId("role-selection-label")).toBeVisible();
		});

		test("should show validation errors for empty fields", async ({ page }) => {
			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Try to submit the form without filling any fields
			await page.getByTestId("add-user-slideover-primary-button").click();

			// Check for validation error messages
			await expect(page.getByTestId("email-error")).toBeVisible();
			await expect(page.getByTestId("email-error")).toContainText("メールアドレスは必須です");

			await expect(page.getByTestId("name-error")).toBeVisible();
			await expect(page.getByTestId("name-error")).toContainText("氏名は必須です");

			await expect(page.getByTestId("roleIds-error")).toBeVisible();
			await expect(page.getByTestId("roleIds-error")).toContainText("ロールを選択してください");
		});

		test("should show validation error for invalid email format", async ({ page }) => {
			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Fill an invalid email
			await page.locator("#email").fill("invalid-email");
			await page.locator("#name").fill("Test User");

			// Submit the form
			await page.getByTestId("add-user-slideover-primary-button").click();

			// Check for email validation error
			await expect(page.getByTestId("email-error")).toBeVisible();
			await expect(page.getByTestId("email-error")).toContainText(
				"メールアドレスの形式が正しくありません"
			);
		});

		test("should select different user roles", async ({ page }) => {
			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Verify role selection area is visible
			await expect(page.getByTestId("role-selection-label")).toBeVisible();
			await expect(page.getByTestId("role-cards-container")).toBeVisible();

			// Initially no roles should be selected
			const adminRoleCard = page.getByTestId("role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			const editorRoleCard = page.getByTestId("role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb");

			await expect(adminRoleCard).toBeVisible();
			await expect(editorRoleCard).toBeVisible();

			// Select admin role
			await adminRoleCard.click();
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "true");

			// Select editor role as well (multiple selection)
			await editorRoleCard.click();
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");

			// Deselect admin role
			await adminRoleCard.click();
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "false");
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");
		});

		test("should successfully invite a new user", async ({ page }) => {
			// Generate a unique email for this test
			const uniqueEmail = `test_user_invitation@example.com`;
			const userName = "Test Invitation User";

			// Check initial pagination count before invitation
			await expect(page.getByTestId("pagination-info")).toContainText("101件中 1 - 50件を表示");

			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Fill the form
			await page.locator("#email").fill(uniqueEmail);
			await page.locator("#name").fill(userName);

			// Select editor role
			const editorRoleCard = page.getByTestId("role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb");
			await editorRoleCard.click();

			// Submit the form and wait for API response
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/users/invite") && response.status() === 200
			);

			await page.getByTestId("add-user-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーを招待しました。");

			// Form should be closed
			await expect(page.getByTestId("add-user-slideover")).not.toBeVisible();

			// Check that pagination count has increased after invitation
			await expect(page.getByTestId("pagination-info")).toContainText("102件中 1 - 50件を表示");

			// Verify the new user appears in the list (might require searching for them)
			await page.locator("#user-search").fill(uniqueEmail);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(uniqueEmail)) &&
					response.status() === 200
			);

			// Check user table shows the new user
			await expect(page.getByTestId("users-table")).toBeVisible();
			const userRows = page.getByTestId(/^user-row-/);
			await expect(userRows).toBeVisible();

			// Check user attributes
			await expect(page.getByText("招待済み")).toBeAttached();
		});

		test("should cancel invitation form", async ({ page }) => {
			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Fill some data
			await page.locator("#email").fill("cancel_test@example.com");
			await page.locator("#name").fill("Cancel Test User");

			// Click the close button
			await page.getByTestId("add-user-slideover-cancel-button").click();

			// Form should be closed and no request should be sent
			await expect(page.getByTestId("add-user-slideover")).not.toBeVisible();
		});
	});
	// Resending Invitation Tests
	test.describe("Resending Invitation", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should show resend button for pending users", async ({ page }) => {
			// Find a pending user (we know the first one on the list is pending)
			const pendingUserId = TestUsersData.enterprise_11.userID;

			// Check that the resend button is present for this user
			await expect(page.getByTestId(`resend-invite-button-${pendingUserId}`)).toBeVisible();

			// Also check that the button contains the expected text
			await expect(page.getByTestId(`resend-invite-button-${pendingUserId}`)).toContainText(
				"招待メールを再送信"
			);
		});

		test("should not show resend button for active users", async ({ page }) => {
			// Search for an active user (Alpha Admin)
			await page.locator("#user-search").fill(TestUsersData.enterprise_1.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_1.email)) &&
					response.status() === 200
			);

			// Check that there's no resend button for this active user
			await expect(
				page.getByTestId(`resend-invite-button-${TestUsersData.enterprise_1.userID}`)
			).not.toBeVisible();
		});

		test("should successfully resend invitation", async ({ page }) => {
			// Find a pending user to resend invitation to
			const pendingUserId = TestUsersData.enterprise_11.userID;

			// Setup an observer for the API request that will be made
			const responsePromise = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${pendingUserId}/resend-invite`) &&
					response.status() === 200
			);

			// Click the resend invitation button
			await page.getByTestId(`resend-invite-button-${pendingUserId}`).click();

			// Wait for the API request to complete
			await responsePromise;

			// Check for success toast message
			await expect(page.getByText("招待メールを送信しました。")).toBeVisible();
		});
	});
	// Role Editing tests
	test.describe("Role Editing", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should open edit role form", async ({ page }) => {
			// Search for the editor user to make sure it's visible
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// Find and click the edit permissions button for the editor user
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_2.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();
			await expect(page.getByTestId("edit-user-slideover-panel")).toBeVisible();

			// Verify that the edit form contains the user's information
			await expect(page.getByTestId("edit-user-title")).toContainText(
				TestUsersData.enterprise_2.name
			);
			await expect(page.getByTestId("edit-user-title")).toContainText(
				TestUsersData.enterprise_2.email
			);

			// Verify the role selector is present
			await expect(page.getByTestId("edit-role-selection-label")).toBeVisible();
			await expect(page.getByTestId("edit-role-cards-container")).toBeVisible();
		});

		test("should change user role from editor to admin", async ({ page }) => {
			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("編集者");

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_2.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Deselect the editor role first (user currently has editor role)
			const editorRoleCard = page.getByTestId(
				"edit-role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			);
			await editorRoleCard.click();

			// Select the admin role
			const adminRoleCard = page.getByTestId("edit-role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			await adminRoleCard.click();

			const editUserResponse = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.enterprise_2.userID}/roles`) &&
					response.status() === 200
			);

			// Submit the form by clicking the save button
			await page.getByTestId("edit-user-slideover-primary-button").click();

			// Wait for API request to complete
			await editUserResponse;

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーのロールを更新しました。");

			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Verify the role shown in the UI is now "管理者" (admin)
			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("管理者");
		});

		test("should change user role from admin to editor", async ({ page }) => {
			// For this test, we'll use a user that's already an admin and change them to editor
			// First, search for the new admin user (we converted in previous test)
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("管理者");

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_2.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Deselect the admin role first (user currently has admin role)
			const adminRoleCard = page.getByTestId("edit-role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			await adminRoleCard.click();

			// Select the editor role
			const editorRoleCard = page.getByTestId(
				"edit-role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			);
			await editorRoleCard.click();

			const editUserResponse = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.enterprise_2.userID}/roles`) &&
					response.status() === 200
			);

			// Submit the form by clicking the save button
			await page.getByTestId("edit-user-slideover-primary-button").click();

			// Wait for API request to complete
			await editUserResponse;

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーのロールを更新しました。");

			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Verify the role shown in the UI is now "編集者" (editor)
			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("編集者");
		});

		test("should cancel role editing without saving changes", async ({ page }) => {
			const searchResponse = page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);
			// Search for the admin user
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await searchResponse;

			// Store the original role text for comparison after cancelation
			const originalRoleText = await page
				.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
				.textContent();

			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("編集者");

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_2.userID}`)
				.click(); // Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Change the role (from editor to admin)
			// Deselect the editor role first
			const editorRoleCard = page.getByTestId(
				"edit-role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			);
			await editorRoleCard.click();

			// Select the admin role
			const adminRoleCard = page.getByTestId("edit-role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			await adminRoleCard.click();

			// Cancel the form by clicking the cancel button
			await page.getByTestId("edit-user-slideover-cancel-button").click();

			// Verify the slideover is no longer visible
			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Verify the role hasn't changed
			const currentRoleText = await page
				.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
				.textContent();
			expect(currentRoleText).toContain(originalRoleText!.trim());
			// Should still be "編集者"
			await expect(
				page.getByTestId(`user-role-${TestUsersData.enterprise_2.userID}`)
			).toContainText("編集者");
		});
	});

	// Multiple Role Selection Tests
	test.describe("Multiple Role Selection", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeAll(async () => {
			// Reset database to clean seed state before multiple role tests
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should invite user with multiple roles", async ({ page }) => {
			// Generate a unique email for this test
			const uniqueEmail = `multi_role_user@example.com`;
			const userName = "Multi Role User";

			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Fill the form
			await page.locator("#email").fill(uniqueEmail);
			await page.locator("#name").fill(userName);

			// Select both admin and editor roles
			const adminRoleCard = page.getByTestId("role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			const editorRoleCard = page.getByTestId("role-card-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb");

			await adminRoleCard.click();
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "true");

			await editorRoleCard.click();
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");

			// Submit the form and wait for API response
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/users/invite") && response.status() === 200
			);

			await page.getByTestId("add-user-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーを招待しました。");

			// Form should be closed
			await expect(page.getByTestId("add-user-slideover")).not.toBeVisible();

			// Search for the new user to verify their roles
			await page.locator("#user-search").fill(uniqueEmail);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(uniqueEmail)) &&
					response.status() === 200
			);

			// Check user table shows the new user
			await expect(page.getByTestId("users-table")).toBeVisible();
			const userRows = page.getByTestId(/^user-row-/);
			await expect(userRows).toBeVisible();

			// Verify the user shows multiple roles in the table
			// Should show the first role name and "他1件" for the additional role
			const userRoleCell = userRows.first().locator('[data-testid*="user-role-"]');
			await expect(userRoleCell).toContainText("管理者");
			await expect(userRoleCell).toContainText("他1件");
		});

		test("should edit user to have multiple roles", async ({ page }) => {
			// Use the suspended_editor user who will have clean state from database reset
			await page.locator("#user-search").fill(TestUsersData.enterprise_16.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_16.email)) &&
					response.status() === 200
			);

			// This user should reliably have only editor role from seed data
			const userRoleCell = page.getByTestId(`user-role-${TestUsersData.enterprise_16.userID}`);
			await expect(userRoleCell).toContainText("閲覧者");

			// Open edit form
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_16.userID}`)
				.click();

			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Editor role should already be selected
			const editorRoleCard = page.getByTestId(
				"edit-role-card-cccccccc-cccc-cccc-cccc-cccccccccccc"
			);
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");

			// Add admin role without removing editor role
			const adminRoleCard = page.getByTestId("edit-role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");
			await adminRoleCard.click();
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "true");

			// Now both roles should be selected
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "true");

			// Submit the form
			const editUserResponse = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.enterprise_16.userID}/roles`) &&
					response.status() === 200
			);

			await page.getByTestId("edit-user-slideover-primary-button").click();
			await editUserResponse;

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーのロールを更新しました。");

			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Verify the user now shows multiple roles in the table
			// Should show the first role name and "他1件" for the additional role
			const updatedUserRoleCell = page.getByTestId(
				`user-role-${TestUsersData.enterprise_16.userID}`
			);
			await expect(updatedUserRoleCell).toContainText("管理者");
			await expect(updatedUserRoleCell).toContainText("他1件");
		});

		test("should remove roles individually from users with multiple roles", async ({ page }) => {
			// Search for the suspended_editor user (who should have multiple roles)
			await page.locator("#user-search").fill(TestUsersData.enterprise_16.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_16.email)) &&
					response.status() === 200
			);

			// Verify user currently shows multiple roles
			const userRoleCell = page.getByTestId(`user-role-${TestUsersData.enterprise_16.userID}`);
			await expect(userRoleCell).toContainText("他1件");

			// Open edit form
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_16.userID}`)
				.click();

			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Both roles should be selected
			const editorRoleCard = page.getByTestId(
				"edit-role-card-cccccccc-cccc-cccc-cccc-cccccccccccc"
			);
			const adminRoleCard = page.getByTestId("edit-role-card-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa");

			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "true");

			// Remove admin role, keep editor role
			await adminRoleCard.click();
			await expect(adminRoleCard).toHaveAttribute("aria-pressed", "false");
			await expect(editorRoleCard).toHaveAttribute("aria-pressed", "true");

			// Submit the form
			const editUserResponse = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.enterprise_16.userID}/roles`) &&
					response.status() === 200
			);

			await page.getByTestId("edit-user-slideover-primary-button").click();
			await editUserResponse;

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーのロールを更新しました。");

			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Verify the user now shows only single role (no "他X件")
			const updatedUserRoleCell = page.getByTestId(
				`user-role-${TestUsersData.enterprise_16.userID}`
			);
			await expect(updatedUserRoleCell).toContainText("閲覧者");
			await expect(updatedUserRoleCell).not.toContainText("他");
		});
	});

	// User Deletion Tests
	test.describe("User Deletion", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.enterprise_1);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should open delete confirmation dialog for active user", async ({ page }) => {
			// Search for the editor user to make sure it's visible
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// Find and click the delete button for the editor user
			await page.getByTestId(`delete-user-button-${TestUsersData.enterprise_2.userID}`).click();

			// Wait for the delete form to be visible
			await expect(page.getByTestId("delete-user-form")).toBeVisible();
			await expect(page.getByTestId("delete-user-slideover-panel")).toBeVisible();

			// Verify that the delete warning contains the user's email
			await expect(page.getByTestId("delete-user-warning")).toContainText(
				TestUsersData.enterprise_2.email
			);

			// Verify the confirmation field is present
			await expect(page.locator("#confirmEmail")).toBeVisible();
		});

		test("should open delete confirmation dialog for pending user (cancel invitation)", async ({
			page
		}) => {
			// Find a pending user
			const pendingUser = TestUsersData.enterprise_11;

			// Ensure the user is visible by searching for them
			await page.locator("#user-search").fill(pendingUser.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(pendingUser.email)) &&
					response.status() === 200
			);

			// Find and click the cancel invite button
			await page.getByTestId(`cancel-invite-button-${pendingUser.userID}`).click();

			// Wait for the delete form to be visible
			await expect(page.getByTestId("delete-user-form")).toBeVisible();
			await expect(page.getByTestId("delete-user-slideover-panel")).toBeVisible();

			// Verify that the delete warning contains the user's email
			await expect(page.getByTestId("delete-user-warning")).toContainText(pendingUser.email);
		});

		test("should show validation error for empty email field", async ({ page }) => {
			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.enterprise_2.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Try to submit without entering email
			await page.getByTestId("delete-user-slideover-primary-button").click();

			// Verify validation error appears
			await expect(page.getByTestId("confirmEmail-error")).toBeVisible();
			await expect(page.getByTestId("confirmEmail-error")).toContainText(
				"メールアドレスの入力は必須です"
			);
		});

		test("should show validation error for incorrect email", async ({ page }) => {
			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.enterprise_2.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Enter incorrect email
			await page.locator("#confirmEmail").fill("wrong@example.com");

			// Try to submit
			await page.getByTestId("delete-user-slideover-primary-button").click();

			// Verify validation error appears
			await expect(page.getByTestId("confirmEmail-error")).toBeVisible();
			await expect(page.getByTestId("confirmEmail-error")).toContainText(
				"メールアドレスが一致しません"
			);
		});

		test("should successfully delete a user", async ({ page }) => {
			// Check initial pagination count before deletion
			const paginationInfo = await page.getByTestId("pagination-info").textContent();
			const initialCount = parseInt(paginationInfo?.match(/(\d+)件中/)?.[1] || "0");

			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.enterprise_2.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Enter the correct email to confirm
			await page.locator("#confirmEmail").fill(TestUsersData.enterprise_2.email);

			// Setup an observer for the API request that will be made
			const responsePromise = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.enterprise_2.userID}`) &&
					response.status() === 200
			);

			// Submit the form
			await page.getByTestId("delete-user-slideover-primary-button").click();

			// Wait for the API request to complete
			await responsePromise;

			// Verify success toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーを削除しました。");

			// Verify the slideover is closed
			await expect(page.getByTestId("delete-user-slideover-panel")).not.toBeVisible();

			// Clear search to view all users and check pagination count decreased
			await page.locator("#user-search").clear();
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					!response.url().includes("search=") &&
					response.status() === 200
			);

			// Check that pagination count has decreased after deletion
			const expectedCount = initialCount - 1;
			await expect(page.getByTestId("pagination-info")).toContainText(
				`${expectedCount}件中 1 - 50件を表示`
			);

			// Verify the user is no longer in the list by searching for them
			await page.locator("#user-search").fill(TestUsersData.enterprise_2.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_2.email)) &&
					response.status() === 200
			);

			// There should be no results
			await expect(page.getByTestId("no-users-message")).toBeVisible();
			await expect(page.getByTestId("no-search-results-message")).toContainText(
				`「${TestUsersData.enterprise_2.email}」に一致するユーザーはありません`
			);
		});

		test("should cancel user deletion without deleting", async ({ page }) => {
			// Search for a user that we're not actually going to delete
			await page.locator("#user-search").fill(TestUsersData.enterprise_16.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_16.email)) &&
					response.status() === 200
			);

			// Verify the user is present
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_16.userID}`)
			).toBeVisible();

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.enterprise_16.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Enter the email (but we will cancel instead of confirming)
			await page.locator("#confirmEmail").fill(TestUsersData.enterprise_16.email);

			// Cancel the deletion by clicking the cancel button
			await page.getByTestId("delete-user-slideover-cancel-button").click();

			// Verify the slideover is closed
			await expect(page.getByTestId("delete-user-slideover-panel")).not.toBeVisible();

			// Verify the user is still in the list
			await expect(
				page.getByTestId(`user-row-${TestUsersData.enterprise_16.userID}`)
			).toBeVisible();
		});

		test("should prevent deletion of current user", async ({ page }) => {
			// Search for the currently logged in user (alpha_admin)
			await page.locator("#user-search").fill(TestUsersData.enterprise_1.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.enterprise_1.email)) &&
					response.status() === 200
			);

			// Verify the user is present
			await expect(page.getByTestId(`user-row-${TestUsersData.enterprise_1.userID}`)).toBeVisible();

			// Verify there is no delete button for the current user
			await expect(
				page.getByTestId(`delete-user-button-${TestUsersData.enterprise_1.userID}`)
			).not.toBeVisible();

			// Also verify there's no edit permissions button for the current user
			await expect(
				page.getByTestId(`edit-permissions-button-${TestUsersData.enterprise_1.userID}`)
			).not.toBeVisible();
		});
	});
});
