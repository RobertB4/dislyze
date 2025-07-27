import { test, expect, APIRequestContext } from "@playwright/test";
import { resetAndSeedDatabase, pool } from "../setup/helpers";
import { TestUsersData } from "../setup/seed";
import { logInAs, logOut } from "../setup/auth";

const IP_WHITELIST_URL = "/settings/ip-whitelist";
const EMERGENCY_DEACTIVATE_URL = "/settings/ip-whitelist/emergency-deactivate";
const MOCK_SENDGRID_API_URL = "http://mock-sendgrid:27000/json";
const MOCK_SENDGRID_API_KEY = "e2e_fake_sendgrid_key";

// Local helper functions for IP whitelist testing

/**
 * Adds an IP whitelist rule directly to the database
 */
async function addIPWhitelistRule(
	tenantId: string,
	ipAddress: string,
	label?: string,
	createdBy = "system"
): Promise<string> {
	const client = await pool.connect();
	try {
		const result = await client.query(
			`INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, label, created_by) 
			 VALUES ($1, $2, $3, $4) 
			 RETURNING id`,
			[tenantId, ipAddress, label || null, createdBy]
		);
		return result.rows[0].id;
	} finally {
		client.release();
	}
}

/**
 * Activates IP whitelist for a tenant by updating enterprise_features
 */
async function activateIPWhitelist(tenantId: string): Promise<void> {
	const client = await pool.connect();
	try {
		await client.query(
			`UPDATE tenants 
			 SET enterprise_features = jsonb_set(
				 enterprise_features, 
				 '{ip_whitelist,active}', 
				 'true'::jsonb
			 ) 
			 WHERE id = $1`,
			[tenantId]
		);
	} finally {
		client.release();
	}
}

/**
 * Deactivates IP whitelist for a tenant by updating enterprise_features
 */
async function deactivateIPWhitelist(tenantId: string): Promise<void> {
	const client = await pool.connect();
	try {
		await client.query(
			`UPDATE tenants 
			 SET enterprise_features = jsonb_set(
				 enterprise_features, 
				 '{ip_whitelist,active}', 
				 'false'::jsonb
			 ) 
			 WHERE id = $1`,
			[tenantId]
		);
	} finally {
		client.release();
	}
}

/**
 * Extracts emergency deactivation token from mock SendGrid emails
 */
async function getEmergencyDeactivationToken(
	request: APIRequestContext,
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
			// Match emergency deactivation URL pattern
			const tokenMatch = htmlBody.match(
				/settings\/ip-whitelist\/emergency-deactivate\?token=([a-zA-Z0-9-_.]+)/
			);
			if (tokenMatch && tokenMatch[1]) {
				return tokenMatch[1];
			}
		} catch (error) {
			console.error("Error fetching or parsing emails from mock-sendgrid:", error);
		}
		if (i < retries - 1) {
			console.log(`Retrying to fetch emergency token, attempt ${i + 1}/${retries}`);
			await new Promise((resolve) => setTimeout(resolve, delay));
		}
	}
	console.error(`Could not find emergency deactivation token after ${retries} retries.`);
	return null;
}

test.describe("IP Whitelist E2E Tests", () => {
	test.beforeAll(async () => {
		await resetAndSeedDatabase();
	});

	test.beforeEach(async ({ page }) => {
		// Set a consistent X-Real-IP header for all tests
		await page.setExtraHTTPHeaders({
			"X-Real-IP": "192.168.1.100"
		});
	});

	test("Add valid IPv4 address", async ({ page, request }) => {
		// Use the IP from beforeEach header
		const testIP = "192.168.1.100";

		// Login as enterprise admin (has ip_whitelist.edit permission)
		await logInAs(page, TestUsersData.enterprise_1);

		// Navigate to IP whitelist page
		await page.goto(IP_WHITELIST_URL);

		// Verify page loads and shows inactive state
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// Add valid IPv4 address
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in the IP address
		await page.getByTestId("ip-address-input").fill(testIP);
		await page.getByTestId("label-input").fill("Test Office IP");

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("IPアドレスを追加しました");

		// Verify IP appears in table
		await expect(page.getByTestId("ip-whitelist-table")).toBeVisible();
		// Check if IP appears in table (looking for the first row data)
		const firstRow = page.locator('[data-testid="ip-whitelist-table-body"] tr').first();
		await expect(firstRow).toBeVisible();
		await expect(firstRow.locator("code")).toContainText(testIP);
	});

	test("Add valid IPv6 address", async ({ page, request }) => {
		const testIPv6 = "2001:0db8:85a3:0000:0000:8a2e:0370:7334";
		const normalizedIPv6 = "2001:db8:85a3::8a2e:370:7334/128"; // Backend normalizes and adds /128

		// Override with IPv6 for this specific test
		await page.setExtraHTTPHeaders({
			"X-Real-IP": testIPv6
		});

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add valid IPv6 address
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in the IPv6 address
		await page.getByTestId("ip-address-input").fill(testIPv6);
		await page.getByTestId("label-input").fill("Test IPv6 Address");

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("IPアドレスを追加しました");

		// Verify IPv6 appears in table (backend normalizes IPv6 and adds /128)
		await expect(
			page.locator('[data-testid="ip-whitelist-table-body"] code', { hasText: normalizedIPv6 })
		).toBeVisible();
	});

	test("Add valid IPv4 CIDR", async ({ page, request }) => {
		const testCIDR = "192.168.1.0/24";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add valid IPv4 CIDR
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in the CIDR
		await page.getByTestId("ip-address-input").fill(testCIDR);
		await page.getByTestId("label-input").fill("Office Network Range");

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("IPアドレスを追加しました");

		// Verify CIDR appears in table
		await expect(
			page.locator('[data-testid="ip-whitelist-table-body"] code', { hasText: testCIDR })
		).toBeVisible();
	});

	test("Add valid IPv6 CIDR", async ({ page, request }) => {
		const testIPv6CIDR = "2001:db8::/32";

		// Override with IPv6 for this specific test
		await page.setExtraHTTPHeaders({
			"X-Real-IP": "2001:0db8:0000:0000:0000:0000:0000:0001"
		});

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add valid IPv6 CIDR
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in the IPv6 CIDR
		await page.getByTestId("ip-address-input").fill(testIPv6CIDR);
		await page.getByTestId("label-input").fill("Corporate IPv6 Network");

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		const toast = page.getByTestId("toast-0");
		await expect(toast).toBeVisible({ timeout: 10000 });
		await expect(toast).toContainText("IPアドレスを追加しました");

		// Verify IPv6 CIDR appears in table
		await expect(
			page.locator('[data-testid="ip-whitelist-table-body"] code', { hasText: testIPv6CIDR })
		).toBeVisible();
	});

	test("Reject invalid IPv4 format", async ({ page, request }) => {
		const invalidIPv4 = "999.999.999.999";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Try to add invalid IPv4 address
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in invalid IPv4 address
		await page.getByTestId("ip-address-input").fill(invalidIPv4);
		await page.getByTestId("label-input").fill("Invalid IP");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Verify specific error message appears
		await expect(page.getByTestId("ip_address-error")).toContainText(
			"IPアドレスの形式が正しくありません"
		);
	});

	test("Reject invalid IPv6 format", async ({ page, request }) => {
		const invalidIPv6 = "2001:invalid:format"; // Invalid characters and format

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Try to add invalid IPv6 address
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in invalid IPv6 address
		await page.getByTestId("ip-address-input").fill(invalidIPv6);
		await page.getByTestId("label-input").fill("Invalid IPv6");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Verify specific error message appears
		await expect(page.getByTestId("ip_address-error")).toContainText(
			"IPアドレスの形式が正しくありません"
		);
	});

	test("Reject invalid CIDR notation", async ({ page, request }) => {
		const invalidCIDR = "192.168.1.0/99"; // /99 is invalid for IPv4 (max is /32)

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Try to add invalid CIDR
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Fill in invalid CIDR
		await page.getByTestId("ip-address-input").fill(invalidCIDR);
		await page.getByTestId("label-input").fill("Invalid CIDR");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Verify specific CIDR range error message appears
		await expect(page.getByTestId("ip_address-error")).toContainText("IPv4のCIDR範囲は0-32です");
	});

	test("Reject empty IP address", async ({ page, request }) => {
		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Try to add empty IP address
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Leave IP address empty, fill label
		await page.getByTestId("label-input").fill("Empty IP Test");

		// Submit the form
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Verify specific required field error message appears
		await expect(page.getByTestId("ip_address-error")).toContainText("IPアドレスは必須です");
	});

	test("Admin can access IP whitelist settings", async ({ page, request }) => {
		// Login as enterprise admin (has all permissions including ip_whitelist.view and ip_whitelist.edit)
		await logInAs(page, TestUsersData.enterprise_1);

		// Navigate to IP whitelist page
		await page.goto(IP_WHITELIST_URL);

		// Verify page loads successfully
		await expect(page.getByTestId("status-section")).toBeVisible();
		await expect(page.getByTestId("status-badge")).toBeVisible();

		// Verify admin has edit permissions - "Add IP" button should be visible
		await expect(page.getByTestId("add-ip-button")).toBeVisible();
		await expect(page.getByTestId("toggle-activation-button")).toBeVisible();
	});

	test("Editor cannot access IP whitelist page", async ({ page, request }) => {
		// Login as enterprise editor (has no ip_whitelist permissions)
		await logInAs(page, TestUsersData.enterprise_2);

		// Try to navigate to IP whitelist page
		await page.goto(IP_WHITELIST_URL);

		// Verify 403 error with permission denied message
		await expect(page.getByTestId("error-message")).toContainText("権限がありません。");
	});

	test("User with view-only permission can see page but not edit", async ({ page, request }) => {
		// First, login as admin to create a role with only ip_whitelist.view permission
		await logInAs(page, TestUsersData.enterprise_1);

		// Navigate to roles management
		await page.goto("/settings/roles");

		// Create new role with view-only permission
		await page.getByTestId("add-role-button").click();
		await expect(page.getByTestId("create-role-slideover-panel")).toBeVisible();

		// Fill in role details (using Input component fields)
		await page.locator('input[name="name"]').fill("IP閲覧者");
		await page.locator('input[name="description"]').fill("IPアドレス制限の閲覧のみ可能");

		// Select only ip_whitelist.view permission using the pill interface
		await page.getByTestId("permission-ip_whitelist-view").click();

		// Save the role
		await page.getByTestId("create-role-slideover-primary-button").click();

		// Verify role was created
		await expect(page.getByTestId("toast-0")).toContainText("ロールを作成しました");

		// Navigate to users management to assign the role
		await page.goto("/settings/users");

		// Find and edit enterprise_3 user (user ID from seed data)
		await page.getByTestId("edit-permissions-button-a0000000-0000-0000-0000-000000000003").click();
		await expect(page.getByTestId("edit-user-slideover-panel")).toBeVisible();

		// Find the newly created role card and click it to select
		const newRoleCardLocator = page
			.locator('[data-testid^="edit-role-card-"]')
			.filter({ hasText: "IP閲覧者" });
		await newRoleCardLocator.click();

		// Save user changes
		await page.getByTestId("edit-user-slideover-primary-button").click();

		// Verify user was updated
		await expect(page.getByTestId("toast-0")).toContainText("ユーザーのロールを更新しました");

		// Logout and login as theƒ user with view-only permission
		await logOut(page);
		await logInAs(page, TestUsersData.enterprise_3);

		// Navigate to IP whitelist page
		await page.goto(IP_WHITELIST_URL);

		// Verify user can see the page (has ip_whitelist.view)
		await expect(page.getByTestId("status-section")).toBeVisible();
		await expect(page.getByTestId("status-badge")).toBeVisible();

		// Verify user cannot see edit buttons (lacks ip_whitelist.edit)
		await expect(page.getByTestId("add-ip-button")).not.toBeVisible();
		await expect(page.getByTestId("toggle-activation-button")).not.toBeVisible();
	});

	test("Multi-tenant isolation verification", async ({ page, request }) => {
		// Login as enterprise tenant 1 admin and add an IP rule
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add IP rule for enterprise tenant 1
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill("10.0.0.1");
		await page.getByTestId("label-input").fill("Enterprise 1 Server");
		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise1 = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise1 = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise1;
		await refreshResponsePromise1;

		// Verify success toast
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");

		// Verify IP appears in table
		await expect(
			page.locator('[data-testid="ip-whitelist-table-body"] code', { hasText: "10.0.0.1" })
		).toBeVisible();

		// Logout and login as internal tenant admin
		await logOut(page);
		await logInAs(page, TestUsersData.internal_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify internal tenant cannot see enterprise tenant's IP rule
		const enterpriseIPLocator = page.locator('[data-testid="ip-whitelist-table-body"] code', {
			hasText: "10.0.0.1"
		});
		await expect(enterpriseIPLocator).not.toBeVisible();

		// Add IP rule for internal tenant to confirm they can add their own
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill("10.0.0.2");
		await page.getByTestId("label-input").fill("Internal Tenant Server");

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise2 = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise2 = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise2;
		await refreshResponsePromise2;

		// Verify internal tenant's IP was added
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");
		await expect(
			page.locator('[data-testid="ip-whitelist-table-body"] code', { hasText: "10.0.0.2" })
		).toBeVisible();

		// Verify enterprise tenant's IP is still not visible
		await expect(enterpriseIPLocator).not.toBeVisible();
	});

	test("Add IP with custom label", async ({ page, request }) => {
		const testIP = "172.16.0.1";
		const customLabel = "カスタムラベルテスト";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add IP with custom label
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(testIP);
		await page.getByTestId("label-input").fill(customLabel);

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");

		// Verify IP appears in table with correct label
		const ipRow = page.locator('[data-testid="ip-whitelist-table-body"] tr').filter({
			has: page.locator("code", { hasText: testIP })
		});
		await expect(ipRow).toBeVisible();
		await expect(ipRow.locator("td").nth(1)).toContainText(customLabel);
	});

	test("Add IP without label", async ({ page, request }) => {
		const testIP = "172.16.0.2";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add IP without label (leave label field empty)
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(testIP);
		// Intentionally leave label-input empty

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify success toast
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");

		// Verify IP appears in table with no label indicator "-"
		const ipRow = page.locator('[data-testid="ip-whitelist-table-body"] tr').filter({
			has: page.locator("code", { hasText: testIP })
		});
		await expect(ipRow).toBeVisible();
		await expect(ipRow.locator("td").nth(1)).toContainText("-");
	});

	test("Edit existing IP label", async ({ page, request }) => {
		const testIP = "172.16.0.3";
		const initialLabel = "初期ラベル";
		const updatedLabel = "更新されたラベル";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// First add IP with initial label
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(testIP);
		await page.getByTestId("label-input").fill(initialLabel);

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify IP was added
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");

		// Find the IP row and click edit label button
		const ipRow = page.locator('[data-testid="ip-whitelist-table-body"] tr').filter({
			has: page.locator("code", { hasText: testIP })
		});
		await expect(ipRow).toBeVisible();
		await expect(ipRow.locator("td").nth(1)).toContainText(initialLabel);

		// Click edit label button for this specific IP
		const editButton = ipRow.locator('[data-testid*="edit-label-button-"]');
		await editButton.click();

		// Verify edit modal opens
		await expect(page.getByTestId("edit-label-slideover-panel")).toBeVisible();

		// Update the label
		await page.getByTestId("edit-label-input").clear();
		await page.getByTestId("edit-label-input").fill(updatedLabel);

		// Wait for the label update API call and subsequent refresh
		const updateResponsePromise = page.waitForResponse(
			(response) =>
				response.url().includes("/api/ip-whitelist/") && response.url().includes("/label/update")
		);
		const refreshResponsePromise2 = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("edit-label-slideover").locator('button[type="submit"]').click();

		// Wait for API calls to complete
		await updateResponsePromise;
		await refreshResponsePromise2;

		// Wait for modal to close (indicates operation completed)
		await expect(page.getByTestId("edit-label-slideover-panel")).not.toBeVisible();

		await expect(ipRow.locator("td").nth(1)).toContainText(updatedLabel);
	});

	test("Clear IP label (set to empty)", async ({ page, request }) => {
		const testIP = "172.16.0.4";
		const initialLabel = "削除予定ラベル";

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// First add IP with initial label
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(testIP);
		await page.getByTestId("label-input").fill(initialLabel);

		// Wait for both the create API call and the subsequent refresh
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Wait for API calls to complete
		await createResponsePromise;
		await refreshResponsePromise;

		// Verify IP was added
		await expect(page.getByTestId("toast-0")).toContainText("IPアドレスを追加しました");

		// Find the IP row and click edit label button
		const ipRow = page.locator('[data-testid="ip-whitelist-table-body"] tr').filter({
			has: page.locator("code", { hasText: testIP })
		});
		await expect(ipRow).toBeVisible();
		await expect(ipRow.locator("td").nth(1)).toContainText(initialLabel);

		// Click edit label button for this specific IP
		const editButton = ipRow.locator('[data-testid*="edit-label-button-"]');
		await editButton.click();

		// Verify edit modal opens
		await expect(page.getByTestId("edit-label-slideover-panel")).toBeVisible();

		// Clear the label field
		await page.getByTestId("edit-label-input").clear();

		// Wait for the label update API call and subsequent refresh
		const updateResponsePromise = page.waitForResponse(
			(response) =>
				response.url().includes("/api/ip-whitelist/") && response.url().includes("/label/update")
		);
		const refreshResponsePromise2 = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("edit-label-slideover").locator('button[type="submit"]').click();

		// Wait for API calls to complete
		await updateResponsePromise;
		await refreshResponsePromise2;

		// Wait for modal to close (indicates operation completed)
		await expect(page.getByTestId("edit-label-slideover-panel")).not.toBeVisible();

		await expect(ipRow.locator("td").nth(1)).toContainText("-");
	});

	test("Duplicate IP address prevention", async ({ page }) => {
		const duplicateIP = "172.16.0.1/32"; // This IP was already added in "Add IP with custom label" test

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify there are existing IPs in the table from previous tests
		const ipRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const initialRowCount = await ipRows.count();
		expect(initialRowCount).toBeGreaterThan(0); // Should have IPs from previous tests

		// Try to add an IP that already exists (172.16.0.1 from "Add IP with custom label" test)
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(duplicateIP);
		await page.getByTestId("label-input").fill("Duplicate IP Attempt");

		// Try to submit - should fail client-side validation
		await page.getByTestId("add-ip-slideover-primary-button").click();

		// Verify validation error appears
		await expect(page.getByTestId("ip_address-error")).toContainText("このIPアドレスは既に登録されています");

		// Verify modal stays open (submission was blocked)
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		// Verify no duplicate IP was added to table (row count unchanged)
		await expect(ipRows).toHaveCount(initialRowCount);

		// Close the modal
		await page.getByTestId("add-ip-slideover-panel").getByRole("button", { name: "キャンセル" }).click();
		await expect(page.getByTestId("add-ip-slideover-panel")).not.toBeVisible();
	});
});

test.describe("Activation/Deactivation Workflows", () => {
	test.beforeEach(async () => {
		// Reset database for each activation/deactivation test
		await resetAndSeedDatabase();
	});

	test("Safe activation (current IP in whitelist)", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify initial inactive state
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// First add current IP range to whitelist to make activation safe
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Docker Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Now perform safe activation - setup listeners before clicking
		const meResponsePromise = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();
		await meResponsePromise;
		await activateResponsePromise;

		// Verify safe activation succeeded without warning modal
		await expect(page.getByTestId("activation-warning-slideover-panel")).not.toBeVisible();

		// Verify status badge changed to active
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Verify toggle button text changed to "無効にする"
		await expect(page.getByTestId("toggle-activation-button")).toContainText("無効にする");
	});

	test("Unsafe activation warning (current IP not in whitelist)", async ({ page }) => {
		const otherIPRange = "10.0.0.0/24"; // Different IP range that won't include Docker network

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify initial inactive state
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// Add a different IP range to whitelist (not current user's IP)
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(otherIPRange);
		await page.getByTestId("label-input").fill("Other Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Now attempt unsafe activation - setup listener before clicking
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;

		// Verify unsafe activation triggers warning modal
		await expect(page.getByTestId("activation-warning-slideover-panel")).toBeVisible();

		// Verify warning modal shows current IP that will be blocked
		await expect(page.getByTestId("activation-warning-alert")).toContainText("172.18.0.");

		// Verify warning message content
		await expect(page.getByTestId("activation-warning-alert")).toContainText(
			"このIPアドレスはIP制限の対象外です"
		);
		await expect(page.getByTestId("activation-warning-alert")).toContainText(
			"アクセスできなくなります"
		);

		// Verify status is still inactive (activation not completed yet)
		await expect(page.getByTestId("status-badge")).toContainText("無効");
	});

	test("Force activation despite warning", async ({ page }) => {
		const otherIPRange = "10.0.0.0/24"; // Different IP range that won't include Docker network

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add a different IP range to whitelist (not current user's IP) to trigger warning
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(otherIPRange);
		await page.getByTestId("label-input").fill("Other Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Trigger unsafe activation warning - setup listener before clicking
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;

		// Verify warning modal appears
		await expect(page.getByTestId("activation-warning-slideover-panel")).toBeVisible();

		// Now force activate despite warning - setup listener before clicking
		const forceActivateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("activation-warning-slideover-primary-button").click();

		await forceActivateResponsePromise;

		// User should now be locked out and redirected to error page
		await expect(page.getByTestId("error-message")).toContainText("権限がありません");
	});

	test("Cancel unsafe activation", async ({ page }) => {
		const otherIPRange = "10.0.0.0/24"; // Different IP range that won't include Docker network

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify initial inactive state
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// Add a different IP range to whitelist (not current user's IP) to trigger warning
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(otherIPRange);
		await page.getByTestId("label-input").fill("Other Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Trigger unsafe activation warning - setup listener before clicking
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;

		// Verify warning modal appears
		await expect(page.getByTestId("activation-warning-slideover-panel")).toBeVisible();

		// Cancel the unsafe activation by clicking cancel button
		await page.getByTestId("activation-warning-slideover-cancel-button").click();

		// Verify modal closes
		await expect(page.getByTestId("activation-warning-slideover-panel")).not.toBeVisible();

		// Verify status remains inactive (activation was cancelled)
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// Verify toggle button text remains "有効にする"
		await expect(page.getByTestId("toggle-activation-button")).toContainText("有効にする");
	});

	test("Safe deactivation", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// First add current IP range and activate IP whitelist
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Docker Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Activate IP whitelist first - setup listeners before clicking
		const meResponsePromise1 = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;
		await meResponsePromise1;

		// Verify IP whitelist is now active
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Now test deactivation - setup listeners before clicking
		const deactivateResponsePromise = page.waitForResponse("/api/ip-whitelist/deactivate");
		const meResponsePromise2 = page.waitForResponse("/api/me");

		// Click deactivate and confirm
		await page.getByTestId("toggle-activation-button").click();
		await expect(page.getByTestId("deactivation-warning-slideover-panel")).toBeVisible();
		await page.getByTestId("deactivation-warning-slideover-primary-button").click();

		await deactivateResponsePromise;
		await meResponsePromise2;

		// Verify deactivation succeeded
		await expect(page.getByTestId("status-badge")).toContainText("無効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("有効にする");
	});

	test("Deactivation confirmation modal", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// First add current IP range and activate IP whitelist
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Docker Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Activate IP whitelist first - setup listeners before clicking
		const meResponsePromise = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;
		await meResponsePromise;

		// Verify IP whitelist is now active
		await expect(page.getByTestId("status-badge")).toContainText("有効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("無効にする");

		// Now test deactivation confirmation modal - click deactivate button
		await page.getByTestId("toggle-activation-button").click();

		// Verify deactivation confirmation modal appears
		await expect(page.getByTestId("deactivation-warning-slideover-panel")).toBeVisible();

		// Verify warning message content
		await expect(page.getByTestId("deactivation-warning-alert")).toContainText(
			"すべてのIPアドレス"
		);
		await expect(page.getByTestId("deactivation-warning-alert")).toContainText(
			"アクセスできるようになります"
		);
		await expect(page.getByTestId("deactivation-warning-alert")).toContainText(
			"本当に無効にしますか"
		);

		// Verify status is still active (deactivation not completed yet)
		await expect(page.getByTestId("status-badge")).toContainText("有効");
	});

	test("Cancel deactivation", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// First add current IP range and activate IP whitelist
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Docker Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Activate IP whitelist first - setup listeners before clicking
		const meResponsePromise = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;
		await meResponsePromise;

		// Verify IP whitelist is now active
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Trigger deactivation confirmation modal
		await page.getByTestId("toggle-activation-button").click();

		// Verify deactivation modal appears
		await expect(page.getByTestId("deactivation-warning-slideover-panel")).toBeVisible();

		// Cancel the deactivation by clicking cancel button
		await page.getByTestId("deactivation-warning-slideover-cancel-button").click();

		// Verify modal closes
		await expect(page.getByTestId("deactivation-warning-slideover-panel")).not.toBeVisible();

		// Verify status remains active (deactivation was cancelled)
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Verify toggle button text remains "無効にする"
		await expect(page.getByTestId("toggle-activation-button")).toContainText("無効にする");
	});

	test("Status badge updates correctly", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify initial inactive state
		await expect(page.getByTestId("status-badge")).toContainText("無効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("有効にする");

		// Add current IP range to whitelist for safe activation
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Docker Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Test activation - status should change from inactive to active
		const meResponsePromise1 = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;
		await meResponsePromise1;

		// Verify status changed to active
		await expect(page.getByTestId("status-badge")).toContainText("有効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("無効にする");

		// Test deactivation - status should change from active back to inactive
		const deactivateResponsePromise = page.waitForResponse("/api/ip-whitelist/deactivate");
		const meResponsePromise2 = page.waitForResponse("/api/me");

		// Click deactivate and confirm
		await page.getByTestId("toggle-activation-button").click();
		await expect(page.getByTestId("deactivation-warning-slideover-panel")).toBeVisible();
		await page.getByTestId("deactivation-warning-slideover-primary-button").click();

		await deactivateResponsePromise;
		await meResponsePromise2;

		// Verify status changed back to inactive
		await expect(page.getByTestId("status-badge")).toContainText("無効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("有効にする");
	});
});

// IP Deletion Safety Tests (3 tests)
test.describe("IP Deletion Safety", () => {
	test.beforeEach(async () => {
		await resetAndSeedDatabase();
	});

	test("Delete IP when whitelist inactive", async ({ page }) => {
		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Verify IP whitelist is initially inactive
		await expect(page.getByTestId("status-badge")).toContainText("無効");

		// Add an IP address to delete
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill("192.168.1.100");
		await page.getByTestId("label-input").fill("Test IP for Deletion");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Verify IP was added to table - get the first row and check its content
		const ipRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const firstRow = ipRows.first();
		await expect(firstRow.getByTestId(/ip-address-/)).toContainText("192.168.1.100");
		await expect(firstRow.getByTestId(/ip-label-/)).toContainText("Test IP for Deletion");

		// Click delete button for the IP we just added
		await firstRow.getByTestId(/delete-ip-button-/).click();

		// Verify delete confirmation modal appears
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Verify warning message shows
		await expect(page.getByTestId("delete-ip-warning")).toContainText("この操作は元に戻せません");
		await expect(page.getByTestId("delete-ip-warning")).toContainText("192.168.1.100");
		await expect(page.getByTestId("delete-ip-warning")).toContainText("削除します");

		// Confirm deletion by typing the IP address
		await page.getByTestId("confirm-ip-input").fill("192.168.1.100/32");

		// Wait for delete API calls
		const deleteResponsePromise = page.waitForResponse(/\/api\/ip-whitelist\/.*\/delete/);
		const refreshAfterDeletePromise = page.waitForResponse("/api/ip-whitelist");

		// Submit delete form
		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		await deleteResponsePromise;
		await refreshAfterDeletePromise;

		// Verify IP was deleted from table (should be empty or not contain our IP)
		const remainingRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const rowCount = await remainingRows.count();
		if (rowCount > 0) {
			// If there are still rows, make sure our deleted IP is not among them
			await expect(
				remainingRows.getByTestId(/ip-address-/).filter({ hasText: "192.168.1.100" })
			).toHaveCount(0);
		}

		// Verify deletion modal is closed
		await expect(page.getByTestId("delete-ip-slideover-panel")).not.toBeVisible();
	});

	test("Prevent deletion of current IP when active", async ({ page }) => {
		const currentIPRange = "172.18.0.0/16"; // CIDR range that covers Docker network IPs

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add current IP range to whitelist
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(currentIPRange);
		await page.getByTestId("label-input").fill("Current Docker Network");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Activate IP whitelist - setup listeners before clicking
		const meResponsePromise = page.waitForResponse("/api/me");
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;
		await meResponsePromise;

		// Verify IP whitelist is now active
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Now try to delete the current IP - get first row (should be our current IP)
		const ipRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const firstRow = ipRows.first();

		// Click delete button for current IP
		await firstRow.getByTestId(/delete-ip-button-/).click();

		// Verify delete confirmation modal appears
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Try to confirm deletion by typing the IP range
		await page.getByTestId("confirm-ip-input").fill(currentIPRange);

		// Wait for delete API call that should fail
		const deleteResponsePromise = page.waitForResponse(
			(response) =>
				response.url().includes("/api/ip-whitelist/") &&
				response.url().includes("/delete") &&
				response.status() >= 400
		);

		// Submit delete form
		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		await deleteResponsePromise;

		// Verify IP was NOT deleted from table (should still be visible)
		const ipRows2 = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const firstRow2 = ipRows2.first();
		await expect(firstRow2.getByTestId(/ip-address-/)).toContainText(currentIPRange);
		await expect(firstRow2.getByTestId(/ip-label-/)).toContainText("Current Docker Network");

		// Verify IP whitelist remains active
		await expect(page.getByTestId("status-badge")).toContainText("有効");

		// Modal should STAY OPEN when deletion fails (not close like in successful deletion)
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();
	});

	test("Confirm deletion with IP address verification", async ({ page }) => {
		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add an IP address that we'll test deletion verification with
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill("10.0.0.1/32");
		await page.getByTestId("label-input").fill("Test Verification IP");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Verify IP was added to table
		const ipRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const firstRow = ipRows.first();
		await expect(firstRow.getByTestId(/ip-address-/)).toContainText("10.0.0.1/32");

		// Click delete button to open confirmation modal
		await firstRow.getByTestId(/delete-ip-button-/).click();

		// Verify delete confirmation modal appears
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Verify warning shows the correct IP address
		await expect(page.getByTestId("delete-ip-warning")).toContainText("10.0.0.1/32");

		// Test 1: Try submitting without entering IP address (should fail validation)
		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		// Modal should stay open due to validation error
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Test 2: Enter wrong IP address (should fail validation)
		await page.getByTestId("confirm-ip-input").fill("192.168.1.1");
		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		// Modal should stay open due to validation error
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Test 3: Enter partial correct IP (should fail validation)
		await page.getByTestId("confirm-ip-input").clear();
		await page.getByTestId("confirm-ip-input").fill("10.0.0.1");
		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		// Modal should stay open due to validation error (missing /32)
		await expect(page.getByTestId("delete-ip-slideover-panel")).toBeVisible();

		// Test 4: Enter correct IP address (should succeed)
		await page.getByTestId("confirm-ip-input").clear();
		await page.getByTestId("confirm-ip-input").fill("10.0.0.1/32");

		// Wait for delete API calls
		const deleteResponsePromise = page.waitForResponse(/\/api\/ip-whitelist\/.*\/delete/);
		const refreshAfterDeletePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("delete-ip-form").getByRole("button", { name: "削除" }).click();

		await deleteResponsePromise;
		await refreshAfterDeletePromise;

		// Verify IP was successfully deleted from table
		const remainingRows = page.getByTestId("ip-whitelist-table-body").locator("tr");
		const rowCount = await remainingRows.count();
		if (rowCount > 0) {
			// If there are still rows, make sure our deleted IP is not among them
			await expect(
				remainingRows.getByTestId(/ip-address-/).filter({ hasText: "10.0.0.1/32" })
			).toHaveCount(0);
		}

		// Verify modal closes after successful deletion
		await expect(page.getByTestId("delete-ip-slideover-panel")).not.toBeVisible();
	});
});

// Emergency Deactivation Tests (3 tests)
test.describe("Emergency Deactivation", () => {
	test.beforeEach(async () => {
		await resetAndSeedDatabase();
	});

	test("Emergency deactivation email and token flow", async ({ page, request }) => {
		const otherIPRange = "10.0.0.0/24"; // Different IP range that won't include Docker network

		// Login as enterprise admin
		await logInAs(page, TestUsersData.enterprise_1);
		await page.goto(IP_WHITELIST_URL);

		// Add a different IP range to whitelist (not current user's IP) to trigger warning
		await page.getByTestId("add-ip-button").click();
		await expect(page.getByTestId("add-ip-slideover-panel")).toBeVisible();

		await page.getByTestId("ip-address-input").fill(otherIPRange);
		await page.getByTestId("label-input").fill("Other Network Range");

		// Wait for create API calls
		const createResponsePromise = page.waitForResponse("/api/ip-whitelist/create");
		const refreshResponsePromise = page.waitForResponse("/api/ip-whitelist");

		await page.getByTestId("add-ip-slideover-primary-button").click();

		await createResponsePromise;
		await refreshResponsePromise;

		// Trigger unsafe activation warning - setup listener before clicking
		const activateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("toggle-activation-button").click();

		await activateResponsePromise;

		// Verify warning modal appears
		await expect(page.getByTestId("activation-warning-slideover-panel")).toBeVisible();

		// Now force activate despite warning - setup listener before clicking
		const forceActivateResponsePromise = page.waitForResponse("/api/ip-whitelist/activate");

		await page.getByTestId("activation-warning-slideover-primary-button").click();

		await forceActivateResponsePromise;

		// User should now be locked out and redirected to error page
		await expect(page.getByTestId("error-message")).toContainText("権限がありません");

		// Verify emergency deactivation email was sent by checking mock SendGrid
		// Wait a bit for email to be processed
		await page.waitForTimeout(2000);

		const emergencyToken = await getEmergencyDeactivationToken(request);
		expect(emergencyToken).toBeTruthy();
		expect(emergencyToken).toMatch(/^[a-zA-Z0-9-_.]+$/); // Valid JWT-like token format

		// Verify user is locked out from all pages, including the top page
		await page.goto("/");
		await expect(page.getByTestId("error-message")).toContainText("権限がありません");

		// Now test the emergency deactivation using the token
		await page.goto(`${EMERGENCY_DEACTIVATE_URL}?token=${emergencyToken}`);

		// Should automatically redirect to IP whitelist settings after successful deactivation
		await expect(page).toHaveURL(IP_WHITELIST_URL);

		// Verify IP whitelist is now deactivated
		await expect(page.getByTestId("status-badge")).toContainText("無効");
		await expect(page.getByTestId("toggle-activation-button")).toContainText("有効にする");
	});

	test("Invalid emergency token shows error", async ({ page }) => {
		// Navigate to emergency deactivate page with invalid token
		await page.goto(`${EMERGENCY_DEACTIVATE_URL}?token=invalid_fake_token_12345`);

		// Should stay on emergency deactivate error page (no redirect)
		await expect(page).toHaveURL(/emergency-deactivate/);

		// Verify error message appears
		await expect(page.locator("h3")).toContainText("緊急解除に失敗しました");
		await expect(page.locator("p")).toContainText("緊急解除リンクが無効または期限切れです");
		await expect(page.locator("p")).toContainText("サポートにお問い合わせください");
	});
});
