import { test, expect } from "@playwright/test";
import { resetAndSeedDatabase, pool } from "$lugia-test/e2e/setup/helpers";
import { TestUsersData } from "$lugia-test/e2e/setup/seed";
import { logInAs } from "$lugia-test/e2e/setup/auth";

test.describe("Settings - Audit Logs Page", () => {
	const auditLogsPageURL = "/settings/audit-logs";

	test.describe("Authentication and Access Control", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test("should redirect to login page when not authenticated", async ({ page }) => {
			await page.goto(auditLogsPageURL);
			await expect(page).toHaveURL(/.*\/auth\/login.*/);
		});

		test("should show audit logs tab for admin with audit_log.view permission", async ({
			page
		}) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto("/settings/profile");
			await expect(page.getByTestId("settings-tab-audit-logs")).toBeVisible();
		});

		test("should not show audit logs tab for editor without audit_log.view permission", async ({
			page
		}) => {
			await logInAs(page, TestUsersData.enterprise_2);
			await page.goto("/settings/profile");
			await expect(page.getByTestId("settings-tab-audit-logs")).not.toBeVisible();
		});

		test("should not show audit logs tab for non-enterprise tenant", async ({ page }) => {
			await logInAs(page, TestUsersData.smb_1);
			await page.goto("/settings/profile");
			await expect(page.getByTestId("settings-tab-audit-logs")).not.toBeVisible();
		});
	});

	test.describe("Viewing Audit Logs", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test.beforeEach(async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(auditLogsPageURL);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();
		});

		test("should display audit logs table with seed data", async ({ page }) => {
			// Seed data has 12 entries, plus login events from logInAs.
			// The table should be visible with rows.
			const rows = page.locator("[data-testid^='audit-log-row-']");
			await expect(rows.first()).toBeVisible();
			expect(await rows.count()).toBeGreaterThan(0);
		});

		test("should display correct table columns", async ({ page }) => {
			const table = page.getByTestId("audit-logs-table");
			await expect(table.locator("th")).toHaveCount(5);
			await expect(table.locator("th").nth(0)).toContainText("日時");
			await expect(table.locator("th").nth(1)).toContainText("操作者");
			await expect(table.locator("th").nth(2)).toContainText("操作");
			await expect(table.locator("th").nth(3)).toContainText("結果");
			await expect(table.locator("th").nth(4)).toContainText("IPアドレス");
		});

		test("should display pagination controls", async ({ page }) => {
			await expect(page.getByTestId("pagination-controls")).toBeVisible();
			await expect(page.getByTestId("pagination-info")).toBeVisible();
			await expect(page.getByTestId("pagination-current")).toBeVisible();
		});

		test("should display filter controls", async ({ page }) => {
			await expect(page.getByTestId("audit-logs-filters")).toBeVisible();
			await expect(page.getByTestId("apply-filters-button")).toBeVisible();
			await expect(page.getByTestId("clear-filters-button")).toBeVisible();
			await expect(page.getByTestId("export-csv-button")).toBeVisible();
		});
	});

	test.describe("Filtering", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test("should filter by resource type", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(auditLogsPageURL);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			// Filter by "auth" resource type (zoroark Select is a custom dropdown, not <select>)
			await page.locator("#resource-type-filter").click();
			await page.getByTestId("resource-type-filter-option-auth").click();
			await page.getByTestId("apply-filters-button").click();

			// Wait for filtered results
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			// URL should contain the filter parameter
			await expect(page).toHaveURL(/resource_type=auth/);

			// All visible rows should be auth events — check the action column contains auth-related labels
			const rows = page.locator("[data-testid^='audit-log-row-']");
			const count = await rows.count();
			expect(count).toBeGreaterThan(0);
			for (let i = 0; i < count; i++) {
				const actionCell = rows.nth(i).locator("td").nth(2);
				await expect(actionCell).toContainText("認証");
			}
		});

		test("should filter by outcome", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(auditLogsPageURL);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			// Filter by failure outcome (zoroark Select is a custom dropdown, not <select>)
			await page.locator("#outcome-filter").click();
			await page.getByTestId("outcome-filter-option-failure").click();
			await page.getByTestId("apply-filters-button").click();

			await expect(page.getByTestId("audit-logs-table")).toBeVisible();
			await expect(page).toHaveURL(/outcome=failure/);

			// All visible rows should show failure badge
			const rows = page.locator("[data-testid^='audit-log-row-']");
			const count = await rows.count();
			expect(count).toBeGreaterThan(0);
			for (let i = 0; i < count; i++) {
				const outcomeCell = rows.nth(i).locator("td").nth(3);
				await expect(outcomeCell).toContainText("失敗");
			}
		});

		test("should clear filters", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			// Start with a filter applied
			await page.goto(`${auditLogsPageURL}?resource_type=auth`);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			// Click clear
			await page.getByTestId("clear-filters-button").click();

			// URL should no longer have filter params
			await expect(page).not.toHaveURL(/resource_type=/);
		});

		test("should show empty state when filter matches nothing", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			// Use a date range in the far future
			await page.goto(`${auditLogsPageURL}?from_date=2099-01-01T00:00:00Z`);

			await expect(page.getByTestId("no-audit-logs-message")).toBeVisible();
		});
	});

	test.describe("Pagination", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test("should navigate between pages", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			// Use small limit to force pagination
			await page.goto(`${auditLogsPageURL}?limit=5`);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			// Should show page 1
			await expect(page.getByTestId("pagination-current")).toContainText("1 /");

			// Click next page
			await page.getByTestId("pagination-next").click();
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();
			await expect(page.getByTestId("pagination-current")).toContainText("2 /");

			// Click previous page
			await page.getByTestId("pagination-prev").click();
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();
			await expect(page.getByTestId("pagination-current")).toContainText("1 /");
		});

		test("should disable prev button on first page", async ({ page }) => {
			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(`${auditLogsPageURL}?limit=5&page=1`);
			await expect(page.getByTestId("audit-logs-table")).toBeVisible();

			await expect(page.getByTestId("pagination-prev")).toBeDisabled();
			await expect(page.getByTestId("pagination-first")).toBeDisabled();
		});
	});

	test.describe("Enterprise Feature Gating", () => {
		test.beforeAll(async () => {
			await resetAndSeedDatabase();
		});

		test("should return 403 when audit_log feature is disabled", async ({ page }) => {
			// Disable audit_log for enterprise tenant
			const client = await pool.connect();
			try {
				await client.query(
					`UPDATE tenants SET enterprise_features = jsonb_set(
						enterprise_features, '{audit_log,enabled}', 'false'
					) WHERE id = $1`,
					[TestUsersData.enterprise_1.tenantID]
				);
			} finally {
				client.release();
			}

			await logInAs(page, TestUsersData.enterprise_1);
			await page.goto(auditLogsPageURL);

			// Should show 403 error
			await expect(page.getByTestId("error-title")).toBeVisible();
			await expect(page.getByTestId("error-title")).toContainText("403");

			// Re-enable for other tests
			const client2 = await pool.connect();
			try {
				await client2.query(
					`UPDATE tenants SET enterprise_features = jsonb_set(
						enterprise_features, '{audit_log,enabled}', 'true'
					) WHERE id = $1`,
					[TestUsersData.enterprise_1.tenantID]
				);
			} finally {
				client2.release();
			}
		});
	});
});
