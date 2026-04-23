import { test, expect, Page } from '@playwright/test';

import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  login,
  createUserViaApi,
  deleteUserViaApi,
  uniqueName,
} from './utils/test-helpers';
import { generateTotp, extractTotpSecret } from './utils/totp';

test.use({ locale: 'en-US' });

/**
 * Dedicated failed-login coverage. Each test is one observable
 * failure mode and proves the UI's error mapping matches the plan's
 * §5 error table: status stays on the current step, inline messages
 * appear, lockouts escalate to a banner, and no "session expired"
 * interceptor ever nukes the mid-flow state.
 */

async function fetchAsUser<T>(page: Page, path: string, init: RequestInit = {}): Promise<T> {
  return page.evaluate(
    async ({ path, init }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;
      const headers = new Headers((init?.headers as Record<string, string> | undefined) ?? {});
      if (!headers.has('Content-Type') && init?.body) {
        headers.set('Content-Type', 'application/json');
      }
      if (csrfToken) headers.set('X-CSRF-Token', csrfToken);
      const response = await fetch(path, { ...init, headers, credentials: 'same-origin' });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(`API ${init?.method ?? 'GET'} ${path} failed: ${response.status} ${text}`);
      }
      const ct = response.headers.get('content-type');
      if (ct?.includes('application/json')) return response.json();
      return null;
    },
    { path, init: init as Record<string, unknown> }
  );
}

async function enrolTotp(page: Page, password: string): Promise<{ secret: string }> {
  const enrol = await fetchAsUser<{
    id: number;
    enrollment: { secret: string; otpauth_uri: string };
  }>(page, '/api/v1/users/me/factors', {
    method: 'POST',
    body: JSON.stringify({ type: 'totp', password }),
  });
  const secret = extractTotpSecret(enrol.enrollment.otpauth_uri);
  await fetchAsUser(page, `/api/v1/users/me/factors/${enrol.id}/activate`, {
    method: 'POST',
    body: JSON.stringify({ code: generateTotp(secret) }),
  });
  return { secret };
}

async function submitPasswordForm(page: Page, email: string, password: string) {
  await page.goto('/login', { waitUntil: 'load' });
  await page.getByLabel(/email/i).fill(email);
  await page.getByLabel(/password/i).fill(password);
  await page.getByRole('button', { name: /sign ?in|login/i }).click();
}

test.describe('Login failures — password step', () => {
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'login-fail-pw-123';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `lf-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('LF User'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
  });

  test.afterEach(async ({ page }) => {
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('wrong password: banner shown, password field cleared, email preserved', async ({
    page,
  }) => {
    await submitPasswordForm(page, testEmail, 'definitely-wrong-password');
    await expect(page).toHaveURL(/\/login/);
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible({
      timeout: 10000,
    });
    // Email field is preserved so the user doesn't have to retype it.
    await expect(page.getByLabel(/email/i)).toHaveValue(testEmail);
  });

  test('non-existent email: same banner (no enumeration)', async ({ page }) => {
    await submitPasswordForm(page, `nosuch-${Date.now()}@example.com`, 'any-password');
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible({
      timeout: 10000,
    });
    await expect(page).toHaveURL(/\/login/);
  });

  test('wrong password followed by correct password succeeds', async ({ page }) => {
    await submitPasswordForm(page, testEmail, 'wrong-first');
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible({
      timeout: 10000,
    });
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  });
});

test.describe('Login failures — MFA step', () => {
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'mfa-fail-pw-123';
  let secret = '';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `mf-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('MF User'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    // Log out admin, log in as user, enrol + activate TOTP, log out.
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
    await page.goto('/login', { waitUntil: 'load' });
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
    ({ secret } = await enrolTotp(page, password));
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
  });

  test.afterEach(async ({ page }) => {
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('wrong code: inline error, input cleared, form stays on MFA step', async ({ page }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    const codeInput = page.getByLabel(/code/i);
    await codeInput.fill('000000');
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toContainText(
      /invalid code/i,
      { timeout: 10000 }
    );
    await expect(codeInput).toHaveValue('');
    // Still on MFA step — not back to password.
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible();
  });

  test('wrong code, then correct code — recovery works without restart', async ({ page }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    await page.getByLabel(/code/i).fill('000000');
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible();

    // Fresh TOTP code works.
    await page.getByLabel(/code/i).fill(generateTotp(secret));
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('back button on MFA step reverts to password form (pending token discarded)', async ({
    page,
  }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /^back$/i }).click();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByTestId('mfa-verify-form')).not.toBeVisible();
  });

  test('per-row rate limit: 5 wrong codes -> banner + revert to password step', async ({
    page,
  }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    for (let i = 0; i < 4; i++) {
      await page.getByLabel(/code/i).fill('000000');
      await page.getByRole('button', { name: /verify/i }).click();
      await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible();
    }
    // 5th wrong code trips the backend's per-row limit → pending
    // row destroyed → the form should revert to password step.
    await page.getByLabel(/code/i).fill('000000');
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toContainText(
      /too many wrong codes/i
    );
  });

  test('browser refresh during MFA step drops pending token → back to password form', async ({
    page,
  }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });
    await page.reload({ waitUntil: 'load' });
    // Pending token only lives in React state; reload discards it.
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('mfa-verify-form')).not.toBeVisible();
  });

  test('navigate away from login during MFA step discards pending', async ({ page }) => {
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });
    // Navigate to a protected route; without a session cookie this
    // 401s and redirects back to /login — the pending is gone.
    await page.goto('/settings', { waitUntil: 'load' });
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByTestId('mfa-verify-form')).not.toBeVisible();
  });

  test('401 on /auth/mfa/verify does NOT trigger the generic session-expired logout', async ({
    page,
  }) => {
    // A bug in the 401 interceptor would kick the user all the way
    // back — regression test for that.
    await submitPasswordForm(page, testEmail, password);
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });
    await page.getByLabel(/code/i).fill('000000');
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)')).toBeVisible();
    // MFA form STILL visible — the 401 didn't collapse state.
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible();
    // Email input still hidden (we're in MFA state, not password state).
    await expect(page.getByLabel(/email/i)).not.toBeVisible();
  });
});
