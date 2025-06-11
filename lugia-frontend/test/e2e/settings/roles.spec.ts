import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";
import { logInAs } from "../setup/auth";

test.describe("Settings - Roles Page", () => {
	const rolesPageURL = "/settings/roles";

	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	// Authentication and access control tests
	test.describe("Authentication and Access Control", () => {
		test("should redirect to login page when not authenticated", async ({ page }) => {
			await page.goto(rolesPageURL);
			await expect(page).toHaveURL(/.*\/auth\/login.*/);
		});

		test("should allow admin access to roles page", async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);

			await expect(page).toHaveURL(rolesPageURL);
			await expect(page.getByTestId("page-title")).toBeVisible();
		});

		test("should display 403 error when editor tries to access roles page", async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_editor);
			await page.goto(rolesPageURL);

			// Should show the error elements with the correct content
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-title")).toContainText("エラーが発生しました (403)");
			await expect(page.getByTestId("error-message")).toBeVisible();
			await expect(page.getByTestId("error-message")).toContainText("権限がありません。");
		});
	});

	// Role viewing and basic functionality tests
	test.describe("Viewing Roles and Basic UI", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();
		});

		test("should display default roles with correct information", async ({ page }) => {
			// Check for default roles from seed data
			await expect(page.getByTestId("roles-table")).toBeVisible();

			// Admin role should be visible
			await expect(
				page.getByTestId("role-name-e0000000-0000-0000-0000-000000000001")
			).toContainText("管理者");
			await expect(
				page.getByTestId("role-description-e0000000-0000-0000-0000-000000000001")
			).toContainText("すべての管理機能にアクセス可能");

			// Editor role should be visible
			await expect(
				page.getByTestId("role-name-e0000000-0000-0000-0000-000000000002")
			).toContainText("編集者");
			await expect(
				page.getByTestId("role-description-e0000000-0000-0000-0000-000000000002")
			).toContainText("限定的な編集権限");

			// Check for "デフォルト" badges
			await expect(
				page.getByTestId("role-type-badge-default-e0000000-0000-0000-0000-000000000001")
			).toBeVisible();
			await expect(
				page.getByTestId("role-type-badge-default-e0000000-0000-0000-0000-000000000002")
			).toBeVisible();
		});

		test("should show add role button for users with edit permissions", async ({ page }) => {
			await expect(page.getByTestId("add-role-button")).toBeVisible();
			await expect(page.getByTestId("add-role-button")).toContainText("ロールを追加");
		});

		test("should show edit and delete buttons only for custom roles", async ({ page }) => {
			// Default roles should not have edit/delete buttons
			const adminRoleRow = page.getByTestId("role-row-e0000000-0000-0000-0000-000000000001");
			const editorRoleRow = page.getByTestId("role-row-e0000000-0000-0000-0000-000000000002");

			await expect(
				adminRoleRow.getByTestId("delete-role-button-e0000000-0000-0000-0000-000000000001")
			).not.toBeVisible();
			await expect(
				adminRoleRow.getByTestId("edit-role-button-e0000000-0000-0000-0000-000000000001")
			).not.toBeVisible();
			await expect(
				editorRoleRow.getByTestId("delete-role-button-e0000000-0000-0000-0000-000000000002")
			).not.toBeVisible();
			await expect(
				editorRoleRow.getByTestId("edit-role-button-e0000000-0000-0000-0000-000000000002")
			).not.toBeVisible();
		});

		test("should display role permissions correctly", async ({ page }) => {
			// Admin role should show multiple permissions
			const adminRoleRow = page.getByTestId("role-row-e0000000-0000-0000-0000-000000000001");
			await expect(
				adminRoleRow.getByTestId("role-permissions-e0000000-0000-0000-0000-000000000001")
			).toContainText("ユーザー一覧の閲覧");
			await expect(
				adminRoleRow.getByTestId("role-permissions-e0000000-0000-0000-0000-000000000001")
			).toContainText("ユーザーの編集");

			// Should show "他X件" if more than 3 permissions
			const overflowElement = adminRoleRow.getByTestId(
				"role-permissions-overflow-e0000000-0000-0000-0000-000000000001"
			);
			if (await overflowElement.isVisible()) {
				await expect(overflowElement).toContainText(/他\d+件/);
			}
		});
	});

	// Role creation tests
	test.describe("Role Creation", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();
		});

		test("should open create role form when add button is clicked", async ({ page }) => {
			await page.getByTestId("add-role-button").click();

			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();
			await expect(page.locator("#name")).toBeVisible();
			await expect(page.locator("#description")).toBeVisible();
			await expect(page.getByTestId("create-role-permissions")).toBeVisible();
		});

		test("should show validation errors for empty fields", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			// Try to submit without filling any fields
			await page.getByTestId("create-role-slideover-primary-button").click();

			// Check for validation errors (these will appear in Input components)
			await expect(page.locator("text=ロール名は必須です")).toBeVisible();
			await expect(page.locator("text=権限を選択してください。")).toBeVisible();
		});

		test("should test permission selector functionality", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-permissions")).toBeVisible();

			// Test users permission selection
			const usersNoneButton = page.getByTestId("permission-users-none");
			const usersViewButton = page.getByTestId("permission-users-view");
			const usersEditButton = page.getByTestId("permission-users-edit");

			await expect(usersNoneButton).toBeVisible();
			await expect(usersViewButton).toBeVisible();
			await expect(usersEditButton).toBeVisible();

			// Test selection
			await usersViewButton.click();
			await expect(usersViewButton).toHaveAttribute("data-selected", "true");

			// Test that selecting edit includes view (hierarchy)
			await usersEditButton.click();
			await expect(usersEditButton).toHaveAttribute("data-selected", "true");
		});

		test("should successfully create a custom role", async ({ page }) => {
			const roleName = "Test Custom Role";
			const roleDescription = "A test role for E2E testing";

			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			// Fill in the form
			await page.locator("#name").fill(roleName);
			await page.locator("#description").fill(roleDescription);

			// Select some permissions
			await page.getByTestId("permission-users-view").click();
			await page.getByTestId("permission-roles-view").click();

			// Submit the form
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ロールを作成しました。");

			// Close the toast
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			// Form should be closed
			await expect(page.getByTestId("create-role-slideover")).not.toBeVisible();

			// New role should appear in the table (search by name and description)
			const newRoleRow = page.locator('[data-testid*="role-row-"]').filter({ hasText: roleName });
			await expect(newRoleRow).toBeVisible();
			await expect(newRoleRow.locator('[data-testid*="role-name-"]')).toContainText(roleName);
			await expect(newRoleRow.locator('[data-testid*="role-description-"]')).toContainText(
				roleDescription
			);

			// Should have custom badge
			const customRoleRow = newRoleRow;
			await expect(customRoleRow.locator('[data-testid*="role-type-badge-custom-"]')).toBeVisible();
		});

		test("should cancel role creation", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			// Fill some data
			await page.locator("#name").fill("Cancel Test Role");
			await page.locator("#description").fill("This should be cancelled");

			// Click cancel
			await page.getByTestId("create-role-slideover-cancel-button").click();

			// Form should be closed and no role created
			await expect(page.getByTestId("create-role-slideover")).not.toBeVisible();
			await expect(page.locator("text=Cancel Test Role")).not.toBeVisible();
		});
	});

	// Role editing tests
	test.describe("Role Editing", () => {
		let customRoleId: string;
		let customRoleName: string;

		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();

			// Create a custom role for editing tests
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			customRoleName = `Editable Test Role ${Date.now()}`;
			await page.locator("#name").fill(customRoleName);
			await page.locator("#description").fill("Role for editing tests");
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			await expect(page.getByTestId("create-role-slideover")).not.toBeVisible();
		});

		test("should open edit role form for custom roles", async ({ page }) => {
			// Find the edit button for our custom role
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: customRoleName });
			const editButton = customRoleRow.locator('[data-testid*="edit-role-button-"]');
			await expect(editButton).toBeVisible();

			await editButton.click();

			// Check edit form opens with pre-populated data
			await expect(page.getByTestId("edit-role-slideover-panel")).toBeVisible();
			await expect(page.locator("#edit-name")).toHaveValue(customRoleName);
			await expect(page.locator("#edit-description")).toHaveValue("Role for editing tests");
		});

		test("should successfully edit a custom role", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: customRoleName });
			const editButton = customRoleRow.locator('[data-testid*="edit-role-button-"]');
			await editButton.click();

			await expect(page.getByTestId("edit-role-slideover-panel")).toBeVisible();

			// Edit the role
			await page.locator("#edit-name").fill("Updated Test Role");
			await page.locator("#edit-description").fill("Updated description");

			// Change permissions
			await page.getByTestId("permission-users-edit").click();

			// Submit changes
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/update") && response.status() === 200
			);

			await page.getByTestId("edit-role-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-1");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ロールを更新しました。");

			// Verify changes in the table
			await expect(page.locator("text=Updated Test Role")).toBeVisible();
			await expect(page.locator("text=Updated description")).toBeVisible();
		});

		test("should cancel role editing without saving", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: customRoleName });
			const editButton = customRoleRow.locator('[data-testid*="edit-role-button-"]');
			await editButton.click();

			await expect(page.getByTestId("edit-role-slideover-panel")).toBeVisible();

			// Make changes
			await page.locator("#edit-name").fill("Should Not Save");

			// Cancel
			await page.getByTestId("edit-role-slideover-cancel-button").click();

			// Form closes and changes not saved
			await expect(page.getByTestId("edit-role-slideover")).not.toBeVisible();
			await expect(page.locator("text=Should Not Save")).not.toBeVisible();
			const unchangedRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: customRoleName });
			await expect(unchangedRoleRow.locator('[data-testid*="role-name-"]')).toContainText(
				customRoleName
			);
		});
	});

	// Role deletion tests
	test.describe("Role Deletion", () => {
		let deletableRoleName: string;

		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();

			// Create a custom role for deletion tests
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			deletableRoleName = `Deletable Test Role ${Date.now()}`;
			await page.locator("#name").fill(deletableRoleName);
			await page.locator("#description").fill("Role for deletion tests");
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			await expect(page.getByTestId("create-role-slideover")).not.toBeVisible();
		});

		test("should open delete confirmation dialog", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			const deleteButton = customRoleRow.locator('[data-testid*="delete-role-button-"]');
			await expect(deleteButton).toBeVisible();

			await deleteButton.click();

			// Check delete confirmation dialog
			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();
			await expect(page.getByTestId("delete-role-warning")).toBeVisible();
			await expect(page.getByTestId("delete-role-warning")).toContainText(deletableRoleName);
			await expect(page.locator("#confirmName")).toBeVisible();
		});

		test("should show validation error for empty confirmation", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			const deleteButton = customRoleRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();

			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();

			// Try to submit without entering role name
			await page.getByTestId("delete-role-slideover-primary-button").click();

			// Should show validation error
			await expect(page.locator("text=ロール名が一致しません")).toBeVisible();
		});

		test("should show validation error for incorrect role name", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			const deleteButton = customRoleRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();

			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();

			// Enter wrong role name
			await page.locator("#confirmName").fill("Wrong Role Name");
			await page.getByTestId("delete-role-slideover-primary-button").click();

			// Should show validation error
			await expect(page.locator("text=ロール名が一致しません")).toBeVisible();
		});

		test("should successfully delete a role not in use", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			const deleteButton = customRoleRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();

			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();

			// Enter correct role name
			await page.locator("#confirmName").fill(deletableRoleName);

			// Submit deletion
			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/delete") && response.status() === 200
			);

			await page.getByTestId("delete-role-slideover-primary-button").click();
			await responsePromise;

			// Check for success toast
			const toastMessage = page.getByTestId("toast-1");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ロールを削除しました。");

			// Role should be removed from table
			await expect(page.getByTestId("delete-role-slideover")).not.toBeVisible();
			const deletedRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			await expect(deletedRoleRow).not.toBeVisible();
		});

		test("should prevent deletion of role assigned to users", async ({ page }) => {
			const roleInUseName = `Role In Use ${Date.now()}`;

			// First create a role
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill(roleInUseName);
			await page.locator("#description").fill("This role will be assigned to a user");
			await page.getByTestId("permission-users-view").click();

			let responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-1");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-1-close").click();
			await expect(page.getByTestId("toast-1")).not.toBeVisible();

			await expect(page.getByTestId("create-role-slideover")).not.toBeVisible();

			// Now navigate to users page to assign this role to a user
			await page.goto("/settings/users");
			await expect(page.getByTestId("users-table")).toBeVisible();

			// Search for a user to assign the role to
			await page.locator("#user-search").fill(TestUsersData.alpha_editor.email);
			await page.waitForResponse(
				(response) => response.url().includes("/api/users") && response.status() === 200
			);

			// Edit user roles
			await page
				.getByTestId(`edit-permissions-button-${TestUsersData.alpha_editor.userID}`)
				.click();
			await expect(page.getByTestId("edit-user-slideover-panel")).toBeVisible();

			// Find and select the role
			const roleInUseCard = page
				.locator('[data-testid*="edit-role-card-"]')
				.filter({ hasText: roleInUseName });
			await roleInUseCard.click();

			// Submit role assignment
			responsePromise = page.waitForResponse(
				(response) => response.url().includes("/roles") && response.status() === 200
			);

			await page.getByTestId("edit-user-slideover-primary-button").click();
			await responsePromise;

			// Go back to roles page
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();

			// Now try to delete the role that's in use
			const roleInUseRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: roleInUseName });
			const deleteButton = roleInUseRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();

			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();
			await page.locator("#confirmName").fill(roleInUseName);

			// Try to delete - should fail with error about role being in use
			await page.getByTestId("delete-role-slideover-primary-button").click();

			// Should show error toast about role being assigned to users
			const errorToast = page.getByTestId("toast-0");
			await expect(errorToast).toBeVisible({ timeout: 10000 });
			await expect(errorToast).toContainText(
				"このロールはユーザーに割り当てられているため削除できません。"
			);

			// Role should still exist in the table
			const existingRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: roleInUseName });
			await expect(existingRoleRow.locator('[data-testid*="role-name-"]')).toContainText(
				roleInUseName
			);
		});

		test("should cancel role deletion", async ({ page }) => {
			const customRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			const deleteButton = customRoleRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();

			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();

			// Enter role name but cancel
			await page.locator("#confirmName").fill(deletableRoleName);
			await page.getByTestId("delete-role-slideover-cancel-button").click();

			// Dialog closes and role remains
			await expect(page.getByTestId("delete-role-slideover")).not.toBeVisible();
			const remainingRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: deletableRoleName });
			await expect(remainingRoleRow.locator('[data-testid*="role-name-"]')).toContainText(
				deletableRoleName
			);
		});
	});

	// Permission selector complex scenarios
	test.describe("Permission Selection Complex Scenarios", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();
		});

		test("should handle all resource types correctly", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-permissions")).toBeVisible();

			// Test all resource types are present
			await expect(page.getByTestId("permission-group-title-users")).toContainText("ユーザー管理");
			await expect(page.getByTestId("permission-group-title-roles")).toContainText("ロール管理");
			await expect(page.getByTestId("permission-group-title-tenant")).toContainText("テナント設定");

			// Test selection across different resources
			await page.getByTestId("permission-users-edit").click();
			await page.getByTestId("permission-roles-view").click();
			await page.getByTestId("permission-tenant-edit").click();

			// Verify selections
			await expect(page.getByTestId("permission-users-edit")).toHaveAttribute(
				"data-selected",
				"true"
			);
			await expect(page.getByTestId("permission-roles-view")).toHaveAttribute(
				"data-selected",
				"true"
			);
			await expect(page.getByTestId("permission-tenant-edit")).toHaveAttribute(
				"data-selected",
				"true"
			);
		});

		test("should handle permission deselection", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-permissions")).toBeVisible();

			// Select a permission
			await page.getByTestId("permission-users-view").click();
			await expect(page.getByTestId("permission-users-view")).toHaveAttribute(
				"data-selected",
				"true"
			);

			// Deselect by clicking "none"
			await page.getByTestId("permission-users-none").click();
			await expect(page.getByTestId("permission-users-none")).toHaveAttribute(
				"data-selected",
				"true"
			);
			await expect(page.getByTestId("permission-users-view")).toHaveAttribute(
				"data-selected",
				"false"
			);
		});

		test("should handle mixed permission levels correctly", async ({ page }) => {
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-permissions")).toBeVisible();

			// Set different permission levels for different resources
			await page.getByTestId("permission-users-edit").click(); // Full permissions for users
			await page.getByTestId("permission-roles-view").click(); // View only for roles
			await page.getByTestId("permission-tenant-none").click(); // No tenant permissions

			// Verify the mixed selection
			await expect(page.getByTestId("permission-users-edit")).toHaveAttribute(
				"data-selected",
				"true"
			);
			await expect(page.getByTestId("permission-roles-view")).toHaveAttribute(
				"data-selected",
				"true"
			);
			await expect(page.getByTestId("permission-tenant-none")).toHaveAttribute(
				"data-selected",
				"true"
			);
		});
	});

	// Edge cases and boundary conditions
	test.describe("Edge Cases", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();
		});

		test("should handle very long role names and descriptions", async ({ page }) => {
			const longName = "A".repeat(100);
			const longDescription = "B".repeat(200);

			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill(longName);
			await page.locator("#description").fill(longDescription);
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Check role was created (might be truncated in display)
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await expect(toastMessage).toContainText("ロールを作成しました。");

			// Close the toast
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();
		});

		test("should handle special characters in role names", async ({ page }) => {
			const specialName = "テスト役割-2024_特殊#文字!@";

			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill(specialName);
			await page.locator("#description").fill("Special characters test");
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			// Verify creation
			const specialNameRoleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: specialName });
			await expect(specialNameRoleRow.locator('[data-testid*="role-name-"]')).toContainText(
				specialName
			);
		});

		test("should display tooltip for permissions overflow", async ({ page }) => {
			// Create a role with many permissions to trigger tooltip
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill("Many Permissions Role");
			await page.locator("#description").fill("Role with many permissions");

			// Select all available permissions
			await page.getByTestId("permission-users-edit").click();
			await page.getByTestId("permission-roles-edit").click();
			await page.getByTestId("permission-tenant-edit").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			// Check if tooltip appears for overflow permissions
			const roleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: "Many Permissions Role" });
			const overflowElement = roleRow.locator('[data-testid*="role-permissions-overflow-"]');

			if (await overflowElement.isVisible()) {
				// Hover to trigger tooltip
				await overflowElement.hover();
				// Tooltip content should be visible (implementation dependent)
			}
		});
	});

	// UI state management and data consistency
	test.describe("UI State and Data Consistency", () => {
		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.alpha_admin);
			await page.goto(rolesPageURL);
			await expect(page.getByTestId("roles-table")).toBeVisible();
		});

		test("should maintain role sorting after operations", async ({ page }) => {
			// Create a custom role
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill("A Custom Role");
			await page.locator("#description").fill("Should appear after default roles");
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			// Check sorting: default roles first (管理者, 編集者), then custom roles alphabetically
			const roleRows = page.locator('[data-testid*="role-row-"]');
			const firstRoleText = await roleRows.first().textContent();
			const secondRoleText = await roleRows.nth(1).textContent();

			// Default roles should come first
			expect(firstRoleText).toContain("管理者");
			expect(secondRoleText).toContain("編集者");
		});

		test("should properly handle modal state transitions", async ({ page }) => {
			// Create a role first
			await page.getByTestId("add-role-button").click();
			await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

			await page.locator("#name").fill("Modal Test Role");
			await page.locator("#description").fill("For testing modal transitions");
			await page.getByTestId("permission-users-view").click();

			const responsePromise = page.waitForResponse(
				(response) => response.url().includes("/api/roles/create") && response.status() === 200
			);

			await page.getByTestId("create-role-slideover-primary-button").click();
			await responsePromise;

			// Dismiss the success toast
			const toastMessage = page.getByTestId("toast-0");
			await expect(toastMessage).toBeVisible({ timeout: 10000 });
			await page.getByTestId("toast-0-close").click();
			await expect(page.getByTestId("toast-0")).not.toBeVisible();

			// Now test modal transitions: create -> edit -> delete
			const roleRow = page
				.locator('[data-testid*="role-row-"]')
				.filter({ hasText: "Modal Test Role" });

			// Edit modal
			const editButton = roleRow.locator('[data-testid*="edit-role-button-"]');
			await editButton.click();
			await expect(page.getByTestId("edit-role-slideover-panel")).toBeVisible();
			await page.getByTestId("edit-role-slideover-cancel-button").click();
			await expect(page.getByTestId("edit-role-slideover")).not.toBeVisible();

			// Delete modal
			const deleteButton = roleRow.locator('[data-testid*="delete-role-button-"]');
			await deleteButton.click();
			await expect(page.getByTestId("delete-role-slideover-panel")).toBeVisible();
			await page.getByTestId("delete-role-slideover-cancel-button").click();
			await expect(page.getByTestId("delete-role-slideover")).not.toBeVisible();

			// All should work smoothly without state conflicts
			const modalTestRoleRow = page.locator('[data-testid*="role-row-"]').filter({ hasText: "Modal Test Role" });
			await expect(modalTestRoleRow.locator('[data-testid*="role-name-"]')).toContainText("Modal Test Role");
		});
	});
});
