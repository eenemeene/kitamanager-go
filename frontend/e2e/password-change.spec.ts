import { test, expect, Page } from '@playwright/test';
import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  login,
  createUserViaApi,
  deleteUserViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

// Log in via the HTTP API without going through the React app. Useful when we
// want the browser to own a session without relying on the login form (which
// has its own flakiness in CI).
async function apiLogin(page: Page, email: string, password: string) {
  await page.goto('/api/v1/health', { waitUntil: 'load' });
  await page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ email, password }),
      });
      if (!response.ok) {
        throw new Error(`Login failed: ${response.status}`);
      }
    },
    { email, password }
  );
}

async function apiLogout(page: Page) {
  await page.evaluate(async () => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const csrfToken = csrfMatch ? csrfMatch[1] : null;
    await fetch('/api/v1/logout', {
      method: 'POST',
      credentials: 'same-origin',
      headers: csrfToken ? { 'X-CSRF-Token': csrfToken } : {},
    });
  });
}

test.describe('Password change flow (regression for #134)', () => {
  // Each test provisions its own user as the superadmin, then signs in as
  // that user. The test user is torn down at the end regardless of outcome.
  let testUserId: number | undefined;
  let testUserEmail = '';
  const originalPassword = 'original-pw-12345';
  const newPassword = 'new-password-98765';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testUserEmail = `pwchange-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('Pwchange User'),
      email: testUserEmail,
      password: originalPassword,
      active: true,
    });
    testUserId = user.id;
    await apiLogout(page);
  });

  test.afterEach(async ({ page }) => {
    if (testUserId !== undefined) {
      // Re-auth as superadmin to delete the test user.
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('user can change password via settings page and log in with new password', async ({
    page,
  }) => {
    // Sign in as the freshly-created test user.
    await apiLogin(page, testUserEmail, originalPassword);
    await page.goto('/settings', { waitUntil: 'load' });

    // Form must render — this proves the page loaded with a valid session.
    await expect(page.getByRole('heading', { name: /change password/i })).toBeVisible({
      timeout: 10000,
    });

    // Fill the change-password form.
    await page.getByLabel(/current password/i).fill(originalPassword);
    await page.getByLabel(/^new password/i).fill(newPassword);
    await page.getByLabel(/confirm new password/i).fill(newPassword);
    await page.getByRole('button', { name: /change password/i }).click();

    // Success toast confirms the 200 from the API.
    await expect(page.getByText(/password changed/i).first()).toBeVisible({ timeout: 10000 });

    // #134 regression: the caller's own session must survive the change.
    // Navigate to a protected page and confirm we are NOT bounced to /login.
    await page.goto('/', { waitUntil: 'load' });
    await expect(page).not.toHaveURL(/.*login/);

    // Now the real regression: log out and log back in with the new password.
    await apiLogout(page);
    await apiLogin(page, testUserEmail, newPassword);
    await page.goto('/', { waitUntil: 'load' });

    // Before the fix, /api/v1/me would 401 here (sentinel row still blocking)
    // and the interceptor would silently bounce the user back to /login.
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 });

    // Old password must no longer work.
    await apiLogout(page);
    await expect(
      apiLogin(page, testUserEmail, originalPassword)
    ).rejects.toThrow(/Login failed: 401/);
  });
});
