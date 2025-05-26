import { test, expect } from '@playwright/test';

test.describe('Auth - Signup Page', () => {
  const signupURL = '/auth/signup';
  const loginURL = '/auth/login';
  const existingUserEmail = 'alpha_admin@example.com'; // From seed.sql
  const uniqueUserEmail = `testuser_${Date.now()}@example.com`;

  test.beforeEach(async ({ page }) => {
    // Optional: Add console/page error listeners for all tests in this describe block
    page.on('console', msg => console.log(`BROWSER CONSOLE (${test.info().title}):`, msg.text()));
    page.on('pageerror', error => {
      console.error(`PAGE ERROR (${test.info().title}):`, error);
    });
    await page.goto(signupURL);
  });

  test('should load the page correctly and link to login', async ({ page }) => {
    await expect(page.locator('h2:has-text("アカウントを作成")')).toBeVisible();
    await expect(page.locator('#company_name')).toBeVisible();
    await expect(page.locator('#user_name')).toBeVisible();
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#password_confirm')).toBeVisible();
    await expect(page.locator('button[type="submit"]:has-text("アカウントを作成")')).toBeVisible();

    await page.locator('a[href="/auth/login"]:has-text("既存のアカウントにログイン")').click();
    await expect(page).toHaveURL(loginURL);
  });

  test('should display error when email already exists', async ({ page, baseURL }) => {
    // Note: page.goto(signupURL) is handled by beforeEach
    console.log(`Navigating to: ${baseURL}${signupURL}`);
      console.log('Current URL:', page.url());
    console.log('Page title:', await page.title());

    const headingLocator = page.locator('h2:has-text("アカウントを作成")');
    try {
      await expect(headingLocator).toBeVisible({ timeout: 15000 });
      console.log('Signup page heading is visible.');
    } catch (e) {
      console.error('Signup page heading not visible or timed out.');
      await page.screenshot({ path: 'test-results/debug-screenshots/signup-page-load-failure.png', fullPage: true });
      throw e;
    }

    const companyNameLocator = page.locator('#company_name');
    try {
      console.log('Waiting for #company_name to be visible...');
      await expect(companyNameLocator).toBeVisible({ timeout: 10000 });
      console.log('#company_name is visible.');
    } catch (e) {
      console.error('#company_name not visible or timed out.');
      await page.screenshot({ path: 'test-results/debug-screenshots/company-name-not-visible.png', fullPage: true });
      throw e;
    }
    
    console.log('Filling #company_name for existing email test...');
    await companyNameLocator.fill('Test Company Existing Email');
    console.log('#company_name filled for existing email test.');

    await page.locator('#user_name').fill('Test User Existing');
    await page.locator('#email').fill(existingUserEmail);
    await page.locator('#password').fill('password123');
    await page.locator('#password_confirm').fill('password123');

    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();

    await expect(page).toHaveURL(signupURL);

    const toastMessage = page.locator('.toast-item.error, [role="alert"], .toast-message');
    try {
      await expect(toastMessage.first()).toBeVisible({ timeout: 10000 });
      console.log('Toast message element is visible for existing email.');
      await expect(toastMessage.first()).toContainText('このメールアドレスは既に使用されています。');
      console.log('Correct toast message text found for existing email.');
    } catch (e) {
      console.error('Toast message not visible or incorrect text for existing email.');
      await page.screenshot({ path: 'test-results/debug-screenshots/toast-error-existing-email.png', fullPage: true });
      throw e;
    }
  });

  test('should display validation errors for all required fields if empty', async ({ page }) => {
    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();
    
    const companyNameErrorLocator = page.locator('#company_name + p.mt-1.text-sm.text-red-600');
    try {
      await expect(companyNameErrorLocator).toContainText('会社名は必須です', { timeout: 7000 });
    } catch (e) {
      console.error('Failed to find company name error or text mismatch.');
      await page.screenshot({ path: 'test-results/debug-screenshots/validation-error-company_name.png', fullPage: true });
      throw e;
    }

    await expect(page.locator('#user_name + p.mt-1.text-sm.text-red-600')).toContainText('氏名は必須です');
    await expect(page.locator('#email + p.mt-1.text-sm.text-red-600')).toContainText('メールアドレスは必須です');
    await expect(page.locator('#password + p.mt-1.text-sm.text-red-600')).toContainText('パスワードは必須です');
    await expect(page.locator('#password_confirm + p.mt-1.text-sm.text-red-600')).toContainText('パスワードを確認してください');
  });

  test('should display validation error for invalid email format', async ({ page }) => {
    await page.locator('#email').fill('invalidemail');
    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();
    await expect(page.locator('#email + p.mt-1.text-sm.text-red-600')).toContainText('メールアドレスの形式が正しくありません');
  });

  test('should display validation error for password too short', async ({ page }) => {
    await page.locator('#password').fill('1234567');
    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();
    await expect(page.locator('#password + p.mt-1.text-sm.text-red-600')).toContainText('パスワードは8文字以上である必要があります');
  });

  test('should display validation error when passwords do not match', async ({ page }) => {
    await page.locator('#password').fill('password123');
    await page.locator('#password_confirm').fill('password456');
    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();
    await expect(page.locator('#password_confirm + p.mt-1.text-sm.text-red-600')).toContainText('パスワードが一致しません');
  });

  test('should allow successful user registration with unique email', async ({ page, baseURL }) => {
    // Listen for /me requests
    page.route('**/me', async route => {
      const request = route.request();
      console.log('INTERCEPTED /me REQUEST:');
      console.log('  URL:', request.url());
      console.log('  METHOD:', request.method());
      const requestHeaders = await request.allHeaders();
      console.log('  HEADERS:', requestHeaders);
      if (requestHeaders['cookie']) {
        console.log('  COOKIE HEADER SENT:', requestHeaders['cookie']);
      } else {
        console.log('  NO COOKIE HEADER SENT with /me request.');
      }
      route.continue();
    });

    // Listen for /auth/signup POST response to inspect Set-Cookie headers
    let signupResponseHeaders = null; // Variable to store headers
    page.on('response', async (response) => {
      if (response.url().includes('/auth/signup') && response.request().method() === 'POST') {
        console.log('INTERCEPTED /auth/signup RESPONSE:');
        console.log('  URL:', response.url());
        console.log('  STATUS:', response.status());
        signupResponseHeaders = await response.allHeaders(); // Store headers
        console.log('  HEADERS:', signupResponseHeaders);
        // Playwright often presents set-cookie as a single string if multiple are set, separated by newline in some contexts
        // or an array if using response.headerValues('set-cookie')
        const setCookieHeader = signupResponseHeaders['set-cookie'] || (await response.headerValues('set-cookie')).join('\n');
        if (setCookieHeader) {
          console.log('  SET-COOKIE HEADER RECEIVED (raw):', setCookieHeader);
        } else {
          console.log('  NO SET-COOKIE HEADER in /auth/signup response.');
        }
      }
    });

    await page.locator('#company_name').fill('New Test Corp');
    await page.locator('#user_name').fill('New Test User');
    await page.locator('#email').fill(uniqueUserEmail);
    await page.locator('#password').fill('validPassword123');
    await page.locator('#password_confirm').fill('validPassword123');

    console.log('Submitting signup form...');
    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();
    console.log('Signup form submitted.');

    // Wait a bit to ensure response listeners have a chance to fire and log
    await page.waitForTimeout(2000); // Increased slightly

    console.log('Waiting for potential post-signup actions (like /me call)...');
    try {
      await page.waitForResponse(response => {
        console.log(`NETWORK RESPONSE: URL: ${response.url()}, Status: ${response.status()}`);
        return response.url().includes('/me') && (response.status() === 200 || response.status() === 401 || response.status() === 403);
      }, { timeout: 10000 });
      console.log('/me response received (or timed out waiting).');
    } catch (e) {
      console.error('Did not see a /me response within timeout or other error waiting for it:', e.message);
    }
    
    const cookiesAfterSignup = await page.context().cookies(); // Get cookies for the baseURL
    console.log('BROWSER COOKIES after signup action and /me call:', cookiesAfterSignup);
    if (cookiesAfterSignup.length === 0) {
        console.log('No cookies found in browser context for this domain.', baseURL);
    }

    console.log('Asserting URL after signup...');
    await expect(page).toHaveURL( /\/$/ , { timeout: 10000 });
    console.log('Successfully redirected to /auth/login after signup.');

    const errorToast = page.locator('.toast-item.error, [role="alert"], .toast-message');
    await expect(errorToast.first()).not.toBeVisible({ timeout: 2000 });
  });

}); 