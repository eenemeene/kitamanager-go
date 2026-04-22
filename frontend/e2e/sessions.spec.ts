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

// Light-weight API login helper — same shape as in password-change.spec.ts.
async function apiLogin(page: Page, email: string, password: string) {
  await page.goto('/api/v1/health', { waitUntil: 'load' });
  await page.evaluate(
    async ({ email, password }) => {
      const r = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ email, password }),
      });
      if (!r.ok) throw new Error(`Login failed: ${r.status}`);
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

test.describe('Active sessions card on /settings', () => {
  let testUserId: number | undefined;
  let testUserEmail = '';
  const password = 'sess-ui-pw-12345';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testUserEmail = `sess-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('SessUser'),
      email: testUserEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    await apiLogout(page);
  });

  test.afterEach(async ({ page }) => {
    if (testUserId !== undefined) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('shows the caller\'s current session with the "this device" badge', async ({ page }) => {
    await apiLogin(page, testUserEmail, password);
    await page.goto('/settings', { waitUntil: 'load' });

    // Sessions card title
    await expect(page.getByRole('heading', { name: /active sessions/i })).toBeVisible({
      timeout: 10000,
    });

    // Current session badge
    await expect(page.getByText(/this device/i).first()).toBeVisible();
  });

  test('user with two sessions can revoke the other one from the UI', async ({ browser, page }) => {
    // Session A: in the main page (will be the "current" one shown on /settings).
    await apiLogin(page, testUserEmail, password);

    // Session B: a second login in an isolated browser context. This creates
    // a second sessions row for the same user.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await apiLogin(pageB, testUserEmail, password);

    // Open settings in page A — two sessions should be listed.
    await page.goto('/settings', { waitUntil: 'load' });
    await expect(page.getByRole('heading', { name: /active sessions/i })).toBeVisible({
      timeout: 10000,
    });

    // The card must show exactly one "This device" badge (session A) and a
    // Revoke button for session B. Wait for the list to populate.
    await expect(page.getByText(/this device/i).first()).toBeVisible();
    const revokeButton = page.getByRole('button', { name: /revoke/i });
    await expect(revokeButton.first()).toBeVisible({ timeout: 10000 });

    // Clicking Revoke opens the confirmation dialog.
    await revokeButton.first().click();
    const dialog = page.getByRole('alertdialog');
    await expect(dialog).toBeVisible();

    // Confirm in the dialog — Playwright finds the Revoke button inside the dialog.
    await dialog.getByRole('button', { name: /revoke/i }).click();

    // Success toast surfaces.
    await expect(page.getByText(/session revoked/i).first()).toBeVisible({ timeout: 10000 });

    // Session B's next authenticated request must 401 because its row is gone.
    const status = await pageB.evaluate(
      async () => (await fetch('/api/v1/me', { credentials: 'same-origin' })).status
    );
    expect(status).toBe(401);

    await ctxB.close();
  });
});
