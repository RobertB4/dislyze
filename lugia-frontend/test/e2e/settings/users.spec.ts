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
		test("should redirect to login page when not authenticated", async ({ page }) => {
			await page.goto(usersPageURL);
			await expect(page).toHaveURL(/.*\/auth\/login.*/);
		});

		test("should allow admin access to users page", async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(usersPageURL);

			await expect(page).toHaveURL(usersPageURL);
			await expect(page.getByRole("heading", { name: "ユーザー管理" })).toBeVisible();
		});

		test("should display 403 error when editor tries to access users page", async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_editor);
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
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block using the helper function
			await logInAs(page, TestUsersData.alpha_admin);

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
				page.getByTestId(
					`user-row-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
			).toBeVisible();
			await expect(
				page.getByTestId(
					`user-name-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
			).toContainText("PendingXT Editor");

			// Check for the Rate Limit Test user (second user in order)
			await expect(
				page.getByTestId(`user-row-${TestUsersData.pending_editor_for_rate_limit_test.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-name-${TestUsersData.pending_editor_for_rate_limit_test.userID}`)
			).toContainText("Pending Editor Rate Limit Test");

			// The first page should only contain 2 users due to pagination
			expect(await page.getByTestId(/^user-row-/).count()).toBe(2);
		});

		test("should search for users by name", async ({ page }) => {
			// Enter search term in the search input using id selector
			await page.locator("#user-search").fill("Alpha Editor");

			// Since we're changing the URL with search, wait for navigation to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes("search=Alpha") &&
					response.status() === 200
			);

			// Verify only matching users are displayed
			await expect(page.getByTestId(`user-row-${TestUsersData.alpha_editor.userID}`)).toBeVisible();

			// User should see their search term in the URL
			await expect(page).toHaveURL(/.*search=Alpha\+Editor.*/);

			// Only the editor user should be visible
			expect(await page.getByTestId(/^user-row-/).count()).toBe(1);
		});

		test("should search for users by email", async ({ page }) => {
			// Enter search term in the search input using id selector
			await page.locator("#user-search").fill("alpha_editor@example.com");

			// Since we're changing the URL with search, wait for navigation to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes("alpha_editor") &&
					response.status() === 200
			);

			// Verify only matching users are displayed
			await expect(page.getByTestId(`user-row-${TestUsersData.alpha_editor.userID}`)).toBeVisible();

			// User should see their search term in the URL
			await expect(page).toHaveURL(/.*search=alpha_editor%40example.com.*/);

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
			const pendingUser = TestUsersData.pending_editor_for_rate_limit_test;
			await expect(page.getByTestId(`user-status-${pendingUser.userID}`)).toBeVisible();

			// For pending users, check the badge color and text
			await expect(page.getByTestId(`user-status-${pendingUser.userID}`)).toContainText("招待済み");
			await expect(page.getByTestId(`user-status-badge-${pendingUser.userID}`)).toHaveClass(
				/bg-yellow-100/
			);

			// Now search for the admin user specifically to test active status
			await page.locator("#user-search").fill(TestUsersData.alpha_admin.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_admin.email)) &&
					response.status() === 200
			);

			// Check active user status
			await expect(
				page.getByTestId(`user-status-${TestUsersData.alpha_admin.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-status-${TestUsersData.alpha_admin.userID}`)
			).toContainText("有効");
			await expect(
				page.getByTestId(`user-status-badge-${TestUsersData.alpha_admin.userID}`)
			).toHaveClass(/bg-green-100/);

			// Clear search and search for suspended user
			await page.locator("#user-search").fill(TestUsersData.suspended_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.suspended_editor.email)) &&
					response.status() === 200
			);

			// Check suspended user status
			await expect(
				page.getByTestId(`user-status-${TestUsersData.suspended_editor.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-status-${TestUsersData.suspended_editor.userID}`)
			).toContainText("停止中");
			await expect(
				page.getByTestId(`user-status-badge-${TestUsersData.suspended_editor.userID}`)
			).toHaveClass(/bg-red-100/);
		});
	});
	// Pagination tests
	test.describe("Pagination", () => {
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.alpha_admin);

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

			// Check pagination info shows correct counts (6 total users, showing 1-2)
			await expect(page.getByTestId("pagination-info")).toContainText("6件中 1 - 2件を表示");

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
			// We're on page 1, which shows users b...6 and b...5
			await expect(
				page.getByTestId(
					`user-row-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
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
			await expect(page.getByTestId("pagination-info")).toContainText("6件中 3 - 4件を表示");

			// Page 2 should show users b...4 and b...3
			await expect(
				page.getByTestId(`user-row-${TestUsersData.suspended_editor.userID}`)
			).toBeVisible();
			await expect(
				page.getByTestId(`user-row-${TestUsersData.pending_editor_valid_token.userID}`)
			).toBeVisible();

			// Users from page 1 should not be visible anymore
			await expect(
				page.getByTestId(
					`user-row-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
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
			await expect(page.getByTestId("pagination-info")).toContainText("6件中 1 - 2件を表示");

			// Page 1 users should be visible again
			await expect(
				page.getByTestId(
					`user-row-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
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

			// Should be on page 3
			await expect(page.getByTestId("pagination-current")).toContainText("3 / 3");
			await expect(page.getByTestId("pagination-info")).toContainText("6件中 5 - 6件を表示");

			// Page 3 should show users b...2 and b...1 (Alpha Editor and Alpha Admin)
			await expect(page.getByTestId(`user-row-${TestUsersData.alpha_editor.userID}`)).toBeVisible();
			await expect(page.getByTestId(`user-row-${TestUsersData.alpha_admin.userID}`)).toBeVisible();

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
				page.getByTestId(
					`user-row-${TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID}`
				)
			).toBeVisible();
		});

		test("should persist search results when navigating through pages", async ({ page }) => {
			const searchResponsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("search=Editor") &&
					response.status() === 200
				);
			});

			// Search for "Editor" which should match multiple users across pages
			await page.locator("#user-search").fill("Editor");

			// Wait for search to complete
			await searchResponsePromise;

			await expect(page.getByTestId("users-table")).toBeVisible();

			// Check that pagination reflects the search results
			await expect(page.getByTestId("pagination-info")).toBeVisible();

			const pageResponsePromise = page.waitForResponse((response) => {
				return (
					response.url().includes("/api/users") &&
					response.url().includes("search=Editor") &&
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
			expect(page.url()).toContain("search=Editor");
			expect(page.url()).toContain("page=2");

			// Should still be showing search results, but for page 2
			expect(await page.getByTestId(/^user-row-/).count()).toBeGreaterThan(0);
		});
	});
	// User Invitation Tests
	test.describe("User Invitation", () => {
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.alpha_admin);

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
			await expect(page.locator("#role")).toBeVisible();
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

			// Check default role selection
			await expect(page.locator("#role")).toContainText("編集者");

			// Open the role dropdown
			await page.locator("#role").click();
			await expect(page.getByTestId("role-list")).toBeAttached();

			// Select admin role
			await page.getByTestId("role-option-admin").click();

			// Verify selection changed
			await expect(page.locator("#role")).toContainText("管理者");
		});

		test("should successfully invite a new user", async ({ page }) => {
			// Generate a unique email for this test
			const uniqueEmail = `test_user_invitation@example.com`;
			const userName = "Test Invitation User";

			// Open the invite form
			await page.getByTestId("add-user-button").click();
			await expect(page.getByTestId("add-user-slideover-panel")).toBeVisible();

			// Fill the form
			await page.locator("#email").fill(uniqueEmail);
			await page.locator("#name").fill(userName);

			// Use default role (editor)

			// Submit the form and wait for API response
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/users/invite") && response.status() === 201
			);

			await page.getByTestId("add-user-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーを招待しました。");

			// Form should be closed
			await expect(page.getByTestId("add-user-slideover")).not.toBeVisible();

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
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.alpha_admin);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should show resend button for pending users", async ({ page }) => {
			// Find a pending user (we know the first one on the list is pending)
			const pendingUserId = TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID;

			// Check that the resend button is present for this user
			await expect(page.getByTestId(`resend-invite-button-${pendingUserId}`)).toBeVisible();

			// Also check that the button contains the expected text
			await expect(page.getByTestId(`resend-invite-button-${pendingUserId}`)).toContainText(
				"招待メールを再送信"
			);
		});

		test("should not show resend button for active users", async ({ page }) => {
			// Search for an active user (Alpha Admin)
			await page.locator("#user-search").fill(TestUsersData.alpha_admin.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_admin.email)) &&
					response.status() === 200
			);

			// Check that there's no resend button for this active user
			await expect(
				page.getByTestId(`resend-invite-button-${TestUsersData.alpha_admin.userID}`)
			).not.toBeVisible();
		});

		test("should successfully resend invitation", async ({ page }) => {
			// Find a pending user to resend invitation to
			const pendingUserId = TestUsersData.pending_editor_tenant_A_for_x_tenant_test.userID;

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
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.alpha_admin);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should open edit role form", async ({ page }) => {
			// Search for the editor user to make sure it's visible
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Find and click the edit permissions button for the editor user
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.alpha_editor.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();
			await expect(page.getByTestId("edit-user-slideover-panel")).toBeVisible();

			// Verify that the edit form contains the user's information
			await expect(page.getByTestId("edit-user-title")).toContainText(
				TestUsersData.alpha_editor.name
			);
			await expect(page.getByTestId("edit-user-title")).toContainText(
				TestUsersData.alpha_editor.email
			);

			// Verify the role selector is present
			await expect(page.locator("#edit-role")).toBeVisible();
		});

		test("should change user role from editor to admin", async ({ page }) => {
			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.alpha_editor.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Select the admin role
			await page.locator("#edit-role").click();
			// Wait for the dropdown to be visible
			await page.getByTestId("edit-role-list").waitFor({ state: "visible" });
			await page.getByTestId("edit-role-option-admin").click();

			// Submit the form by clicking the save button
			await page.getByTestId("edit-user-slideover-primary-button").click();

			// Wait for API request to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.alpha_editor.userID}/permissions`) &&
					response.status() === 200
			);

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーの役割を更新しました。");

			// Verify the user's role was updated in the UI
			// First, we need to ensure the page is refreshed/updated with new data
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Verify the role shown in the UI is now "管理者" (admin)
			await expect(
				page.getByTestId(`user-role-${TestUsersData.alpha_editor.userID}`)
			).toContainText("管理者");
		});

		test("should change user role from admin to editor", async ({ page }) => {
			// For this test, we'll use a user that's already an admin and change them to editor
			// First, search for the new admin user (we converted in previous test)
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.alpha_editor.userID}`)
				.click();

			// Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Select the editor role
			await page.locator("#edit-role").click();
			// Wait for the dropdown to be visible
			await page.getByTestId("edit-role-list").waitFor({ state: "visible" });
			await page.getByTestId("edit-role-option-editor").click();

			// Submit the form by clicking the save button
			await page.getByTestId("edit-user-slideover-primary-button").click();

			// Wait for API request to complete
			await page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.alpha_editor.userID}/permissions`) &&
					response.status() === 200
			);

			// Verify toast message appears
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ユーザーの役割を更新しました。");

			// Verify the user's role was updated in the UI
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Verify the role shown in the UI is now "編集者" (editor)
			await expect(
				page.getByTestId(`user-role-${TestUsersData.alpha_editor.userID}`)
			).toContainText("編集者");
		});

		test("should cancel role editing without saving changes", async ({ page }) => {
			// Search for the admin user
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Store the original role text for comparison after cancelation
			const originalRoleText = await page
				.getByTestId(`user-role-${TestUsersData.alpha_editor.userID}`)
				.textContent();

			// Find and click the edit permissions button
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.alpha_editor.userID}`)
				.click(); // Wait for the edit form to be visible
			await expect(page.getByTestId("edit-user-form")).toBeVisible();

			// Change the role (from admin to editor)
			// First click the select to open the dropdown
			await page.locator("#edit-role").click();
			// Wait for the dropdown to be visible
			await page.getByTestId("edit-role-list").waitFor({ state: "visible" });
			await page.getByTestId("edit-role-option-editor").click();

			// Cancel the form by clicking the cancel button
			await page.getByTestId("edit-user-slideover-cancel-button").click();

			// Verify the slideover is no longer visible
			await expect(page.getByTestId("edit-user-slideover-panel")).not.toBeVisible();

			// Search again to ensure the page is refreshed
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Verify the role hasn't changed
			const currentRoleText = await page
				.getByTestId(`user-role-${TestUsersData.alpha_editor.userID}`)
				.textContent();
			expect(currentRoleText).toContain(originalRoleText!.trim());
			// Should still be "管理者" (admin)
			await expect(
				page.getByTestId(`user-role-${TestUsersData.alpha_editor.userID}`)
			).toContainText("編集者");
		});
	});
	// User Deletion Tests
	test.describe("User Deletion", () => {
		test.beforeEach(async ({ page }) => {
			// Login as admin for all tests in this describe block
			await logInAs(page, TestUsersData.alpha_admin);

			// Navigate to users page
			await page.goto(usersPageURL);

			// Wait for users to be loaded
			await expect(page.getByTestId("users-table")).toBeVisible();
		});

		test("should open delete confirmation dialog for active user", async ({ page }) => {
			// Search for the editor user to make sure it's visible
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Find and click the delete button for the editor user
			await page.getByTestId(`delete-user-button-${TestUsersData.alpha_editor.userID}`).click();

			// Wait for the delete form to be visible
			await expect(page.getByTestId("delete-user-form")).toBeVisible();
			await expect(page.getByTestId("delete-user-slideover-panel")).toBeVisible();

			// Verify that the delete warning contains the user's email
			await expect(page.getByTestId("delete-user-warning")).toContainText(
				TestUsersData.alpha_editor.email
			);

			// Verify the confirmation field is present
			await expect(page.locator("#confirmEmail")).toBeVisible();
		});

		test("should open delete confirmation dialog for pending user (cancel invitation)", async ({
			page
		}) => {
			// Find a pending user
			const pendingUser = TestUsersData.pending_editor_tenant_A_for_x_tenant_test;

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
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.alpha_editor.userID}`).click();
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
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.alpha_editor.userID}`).click();
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
			// Search for the editor user
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.alpha_editor.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Enter the correct email to confirm
			await page.locator("#confirmEmail").fill(TestUsersData.alpha_editor.email);

			// Setup an observer for the API request that will be made
			const responsePromise = page.waitForResponse(
				(response) =>
					response.url().includes(`/api/users/${TestUsersData.alpha_editor.userID}`) &&
					response.status() === 204
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

			// Verify the user is no longer in the list
			// Search again for the deleted user
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_editor.email)) &&
					response.status() === 200
			);

			// There should be no results
			await expect(page.getByTestId("no-users-message")).toBeVisible();
			await expect(page.getByTestId("no-search-results-message")).toContainText(
				`「${TestUsersData.alpha_editor.email}」に一致するユーザーはありません`
			);
		});

		test("should cancel user deletion without deleting", async ({ page }) => {
			// Search for a user that we're not actually going to delete
			await page.locator("#user-search").fill(TestUsersData.suspended_editor.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.suspended_editor.email)) &&
					response.status() === 200
			);

			// Verify the user is present
			await expect(
				page.getByTestId(`user-row-${TestUsersData.suspended_editor.userID}`)
			).toBeVisible();

			// Open the delete confirmation dialog
			await page.getByTestId(`delete-user-button-${TestUsersData.suspended_editor.userID}`).click();
			await expect(page.getByTestId("delete-user-form")).toBeVisible();

			// Enter the email (but we will cancel instead of confirming)
			await page.locator("#confirmEmail").fill(TestUsersData.suspended_editor.email);

			// Cancel the deletion by clicking the cancel button
			await page.getByTestId("delete-user-slideover-cancel-button").click();

			// Verify the slideover is closed
			await expect(page.getByTestId("delete-user-slideover-panel")).not.toBeVisible();

			// Verify the user is still in the list
			await expect(
				page.getByTestId(`user-row-${TestUsersData.suspended_editor.userID}`)
			).toBeVisible();
		});

		test("should prevent deletion of current user", async ({ page }) => {
			// Search for the currently logged in user (alpha_admin)
			await page.locator("#user-search").fill(TestUsersData.alpha_admin.email);
			await page.waitForResponse(
				(response) =>
					response.url().includes("/api/users") &&
					response.url().includes(encodeURIComponent(TestUsersData.alpha_admin.email)) &&
					response.status() === 200
			);

			// Verify the user is present
			await expect(page.getByTestId(`user-row-${TestUsersData.alpha_admin.userID}`)).toBeVisible();

			// Verify there is no delete button for the current user
			await expect(
				page.getByTestId(`delete-user-button-${TestUsersData.alpha_admin.userID}`)
			).not.toBeVisible();

			// Also verify there's no edit permissions button for the current user
			await expect(
				page.getByTestId(`edit-permissions-button-${TestUsersData.alpha_admin.userID}`)
			).not.toBeVisible();
		});
	});
});
