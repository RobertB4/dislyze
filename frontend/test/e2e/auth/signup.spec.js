import { test, expect } from '@playwright/test';

test.describe('Auth - Signup Page', () => {
  const signupURL = '/auth/signup';
  const existingUserEmail = 'alpha_admin@example.com'; // From seed.sql

  test('should display error when email already exists', async ({ page, baseURL }) => {
    page.on('console', msg => console.log('BROWSER CONSOLE:', msg.text()));
    page.on('pageerror', error => {
      console.error('PAGE ERROR:', error);
    });

    console.log(`Navigating to: ${baseURL}${signupURL}`);
    await page.goto(signupURL);

    console.log('Current URL:', page.url());
    console.log('Page title:', await page.title());

    // Check for a key static element to ensure page is somewhat loaded
    const headingLocator = page.locator('h2:has-text("アカウントを作成")');
    try {
      await expect(headingLocator).toBeVisible({ timeout: 15000 }); // Increased timeout slightly
      console.log('Signup page heading is visible.');
    } catch (e) {
      console.error('Signup page heading not visible or timed out.');
      // Capture screenshot if heading is not visible, as page might be wrong
      await page.screenshot({ path: 'test-results/debug-screenshots/signup-page-load-failure.png', fullPage: true });
      throw e; // Re-throw the error to fail the test
    }

    // Wait for the specific form element to be visible before trying to fill it
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
    
    console.log('Filling #company_name...');
    await companyNameLocator.fill('Test Company Existing Email');
    console.log('#company_name filled.');

    await page.locator('#user_name').fill('Test User Existing');
    await page.locator('#email').fill(existingUserEmail);
    await page.locator('#password').fill('password123');
    await page.locator('#password_confirm').fill('password123');

    await page.locator('button[type="submit"]:has-text("アカウントを作成")').click();

    // User should remain on the signup page
    await expect(page).toHaveURL(signupURL);

    const toastMessage = page.locator('.toast-item.error, [role="alert"], .toast-message'); // Adjusted selector slightly
    // It's better to wait for the toast to appear first, then check its text
    try {
      await expect(toastMessage.first()).toBeVisible({ timeout: 10000 }); // Wait for the first matching toast
      console.log('Toast message element is visible.');
      await expect(toastMessage.first()).toContainText('このメールアドレスは既に使用されています。');
      console.log('Correct toast message text found.');
    } catch (e) {
      console.error('Toast message not visible or incorrect text.');
      await page.screenshot({ path: 'test-results/debug-screenshots/toast-error.png', fullPage: true });
      throw e;
    }
  });
}); 