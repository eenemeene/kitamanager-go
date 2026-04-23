import { test, expect, Page } from '@playwright/test';

import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  login,
  loginViaForm,
  createUserViaApi,
  deleteUserViaApi,
  uniqueName,
} from './utils/test-helpers';
import {
  addVirtualAuthenticator,
  removeVirtualAuthenticator,
  setUserVerified,
  type VirtualAuthenticator,
} from './utils/webauthn';

test.use({ locale: 'en-US' });

/**
 * End-to-end coverage for the WebAuthn / FIDO2 factor type. All
 * tests run against the real dev server with a real virtual
 * authenticator attached via Chromium's CDP `WebAuthn` domain; the
 * browser generates real ES256 keys and signs real attestations,
 * which the Go backend verifies against the origin + challenge +
 * rpId we configured it with. No mocks.
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

async function logout(page: Page) {
  await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' }).catch(() => {});
  await page.goto('/login', { waitUntil: 'load' });
  await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
}

test.describe('WebAuthn enrolment via Settings', () => {
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'webauthn-e2e-pw-123';
  let authenticator: VirtualAuthenticator | undefined;

  test.beforeEach(async ({ page, context }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `webauthn-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('WebAuthn User'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
    authenticator = await addVirtualAuthenticator(context, page);
  });

  test.afterEach(async ({ page }) => {
    if (authenticator) {
      await removeVirtualAuthenticator(authenticator);
      authenticator = undefined;
    }
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('enrol a security key from Settings', async ({ page }) => {
    await loginViaForm(page, testEmail, password);
    await page.goto('/settings', { waitUntil: 'load' });

    // Card is in the "disabled" state; both Enable (TOTP) and
    // Add-security-key should be visible.
    const addKey = page.getByRole('button', { name: /add security key/i });
    await expect(addKey).toBeVisible({ timeout: 10000 });
    await addKey.click();

    const dialog = page.getByRole('dialog', { name: /add security key/i });
    await expect(dialog).toBeVisible();

    // Password step.
    await dialog.getByLabel(/current password/i).fill(password);
    await dialog.getByRole('button', { name: /continue/i }).click();

    // Chromium's virtual authenticator auto-confirms the prompt, so
    // navigator.credentials.create() resolves without user action.
    // Activation + backup-codes dialog follows.
    const backupDialog = page.getByTestId('backup-codes-dialog');
    await expect(backupDialog).toBeVisible({ timeout: 15000 });
    const codes = await backupDialog.locator('[data-testid^=backup-code-]').allTextContents();
    expect(codes.length).toBeGreaterThan(4);
    await backupDialog.getByRole('checkbox', { name: /saved these codes/i }).check();
    await backupDialog.getByRole('button', { name: /done/i }).click();

    // The factor list now includes a Security Key row.
    await expect(page.getByTestId('factor-row-webauthn')).toBeVisible();
    await expect(page.getByTestId('factor-row-backup_codes')).toBeVisible();
  });
});

test.describe('WebAuthn login', () => {
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'webauthn-login-pw-123';
  let authenticator: VirtualAuthenticator | undefined;

  test.beforeEach(async ({ page, context }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `wa-login-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('WebAuthn Login'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });

    // Enrol the user via the UI so we're in a known "has webauthn"
    // state before each test. The test helper above drives the
    // Settings page; we reproduce it inline to keep each test
    // self-contained.
    authenticator = await addVirtualAuthenticator(context, page);
    await loginViaForm(page, testEmail, password);
    await page.goto('/settings', { waitUntil: 'load' });
    await page.getByRole('button', { name: /add security key/i }).click();
    const dialog = page.getByRole('dialog', { name: /add security key/i });
    await dialog.getByLabel(/current password/i).fill(password);
    await dialog.getByRole('button', { name: /continue/i }).click();
    const backupDialog = page.getByTestId('backup-codes-dialog');
    await expect(backupDialog).toBeVisible({ timeout: 15000 });
    await backupDialog.getByRole('checkbox', { name: /saved these codes/i }).check();
    await backupDialog.getByRole('button', { name: /done/i }).click();
    await expect(page.getByTestId('factor-row-webauthn')).toBeVisible();
    await logout(page);
  });

  test.afterEach(async ({ page }) => {
    if (authenticator) {
      await removeVirtualAuthenticator(authenticator);
      authenticator = undefined;
    }
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {});
      testUserId = undefined;
    }
  });

  test('login completes with WebAuthn assertion', async ({ page }) => {
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();

    const mfaForm = page.getByTestId('mfa-verify-form');
    await expect(mfaForm).toBeVisible({ timeout: 10000 });

    // "Use security key" button — the virtual authenticator
    // auto-confirms the navigator.credentials.get() prompt.
    const useKey = mfaForm.getByRole('button', { name: /use security key/i });
    await expect(useKey).toBeVisible();
    await useKey.click();

    await expect(page).not.toHaveURL(/\/login/, { timeout: 15000 });
  });

  test('cancelled prompt surfaces an inline error and keeps the user on the form', async ({
    page,
  }) => {
    // Flip the UV gesture to "not verified" so the authenticator
    // rejects the ceremony with NotAllowedError-equivalent.
    if (authenticator) {
      await setUserVerified(authenticator, false);
    }

    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();

    const mfaForm = page.getByTestId('mfa-verify-form');
    await expect(mfaForm).toBeVisible({ timeout: 10000 });
    await mfaForm.getByRole('button', { name: /use security key/i }).click();

    // Inline error appears; still on MFA form.
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)').first()).toBeVisible(
      { timeout: 15000 }
    );
    await expect(mfaForm).toBeVisible();

    // Flip UV back on; next click should complete successfully.
    if (authenticator) {
      await setUserVerified(authenticator, true);
    }
    await mfaForm.getByRole('button', { name: /use security key/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 15000 });
  });
});
