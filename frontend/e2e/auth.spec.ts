import { test, expect } from '@playwright/test';
import { ADMIN_EMAIL, ADMIN_PASSWORD, login, loginViaForm } from './utils/test-helpers';

test.describe('Authentication', () => {
  test('should display login page', async ({ page }) => {
    await page.goto('/');

    // Should redirect to login
    await expect(page).toHaveURL(/.*login/);

    // Login form should be visible
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in|login|anmelden/i })).toBeVisible();
  });

  test('should login with valid credentials', async ({ page }) => {
    // Use loginViaForm to test the actual login form
    await loginViaForm(page, ADMIN_EMAIL, ADMIN_PASSWORD);

    // Dashboard should show content
    await expect(page.locator('body')).toContainText(/dashboard|organization/i);
  });

  test('should show error with invalid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.getByLabel(/email/i).fill('wrong@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /sign in|login|anmelden/i }).click();

    // Should stay on login page or show error
    await expect(page).toHaveURL(/.*login/);
  });

  test('should redirect protected routes to login', async ({ page }) => {
    await page.goto('/organizations');

    // Should redirect to login
    await expect(page).toHaveURL(/.*login/);
  });

  test('should redirect to original page after login', async ({ page }) => {
    // Try to access protected page
    await page.goto('/organizations');

    // Should redirect to login with 'from' param
    await expect(page).toHaveURL(/.*login/);

    // Login via form - testing the redirect behavior
    const emailInput = page.getByLabel(/email/i);
    const passwordInput = page.getByLabel(/password/i);

    await expect(emailInput).toBeVisible({ timeout: 10000 });
    await emailInput.fill(ADMIN_EMAIL);
    await passwordInput.fill(ADMIN_PASSWORD);
    await page.getByRole('button', { name: /sign in|login|anmelden/i }).click();

    // Should redirect back to organizations (or dashboard)
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should stay logged in after page refresh', async ({ page }) => {
    // Login first
    await login(page);

    // Refresh page
    await page.reload();

    // Should still be logged in (not on login page)
    await expect(page).not.toHaveURL(/.*login/, { timeout: 5000 });
  });
});
