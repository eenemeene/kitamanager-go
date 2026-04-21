import { test, expect, Page } from '@playwright/test';
import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  login,
  createUserViaApi,
  deleteUserViaApi,
  uniqueName,
  createTestOrg,
  deleteTestOrg,
  addUserToOrgViaApi,
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

// Regression pair for the above: a password change must not silently drop
// the user's org role. Token revocation (part of ChangePassword) invalidates
// the old session, but the user_organizations rows must survive so the next
// login still resolves the role through Casbin. If a future refactor moves
// role data onto the token claim alone, this test fails.
test.describe('Password change preserves org membership', () => {
  let testOrgId: number;
  let testUserId: number | undefined;
  let testUserEmail = '';
  const originalPassword = 'original-membership-pw-12345';
  const newPassword = 'rotated-membership-pw-98765';

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const org = await createTestOrg(page, 'PwMembership');
    testOrgId = org.orgId;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await deleteTestOrg(page, testOrgId).catch(() => {});
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testUserEmail = `pwmembership-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('PwMembership User'),
      email: testUserEmail,
      password: originalPassword,
      active: true,
    });
    testUserId = user.id;
    await addUserToOrgViaApi(page, user.id, testOrgId, 'manager');
    await apiLogout(page);
  });

  test.afterEach(async ({ page }) => {
    if (testUserId !== undefined) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('manager role survives a self-service password change', async ({ page }) => {
    // Sign in as the new manager and confirm their org access resolves
    // BEFORE the password change — gives us a baseline to diff against.
    await apiLogin(page, testUserEmail, originalPassword);
    const preChange = await page.evaluate(
      async (id) =>
        (await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' })).status,
      testOrgId
    );
    expect(preChange).toBe(200);

    // Change password via the settings page (mirrors the existing #134 test
    // so we exercise the same code path a real user hits).
    await page.goto('/settings', { waitUntil: 'load' });
    await expect(page.getByRole('heading', { name: /change password/i })).toBeVisible({
      timeout: 10000,
    });
    await page.getByLabel(/current password/i).fill(originalPassword);
    await page.getByLabel(/^new password/i).fill(newPassword);
    await page.getByLabel(/confirm new password/i).fill(newPassword);
    await page.getByRole('button', { name: /change password/i }).click();
    await expect(page.getByText(/password changed/i).first()).toBeVisible({ timeout: 10000 });

    // Sign out and sign back in with the new password.
    await apiLogout(page);
    await apiLogin(page, testUserEmail, newPassword);

    // Role must still resolve — the user_organizations row is identity-scoped,
    // not token-scoped, so password rotation must not affect it.
    const postChange = await page.evaluate(
      async (id) =>
        (await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' })).status,
      testOrgId
    );
    expect(postChange).toBe(200);

    // And confirm /me still shows the right identity, not a stale session.
    const me = await page.evaluate(async () => {
      const r = await fetch('/api/v1/me', { credentials: 'same-origin' });
      return r.ok ? ((await r.json()) as { email: string }) : null;
    });
    expect(me?.email).toBe(testUserEmail);
  });
});
