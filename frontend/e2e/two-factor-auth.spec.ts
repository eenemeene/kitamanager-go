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
import { generateTotp, extractTotpSecret } from './utils/totp';

test.use({ locale: 'en-US' });

/**
 * End-to-end lifecycle test for 2FA. Everything below is exercised
 * against the running dev server — no mocks, no MSW, no fixtures.
 * TOTP codes are generated from the real base32 secret returned by
 * the enrolment endpoint; backup codes used here are the real ones
 * the activation response hands back.
 *
 * Flow:
 *   1. Superadmin creates a regular user (via API).
 *   2. User logs in with password — no MFA yet, direct to dashboard.
 *   3. User visits Settings and enables 2FA: password → QR → code →
 *      backup codes.
 *   4. User logs out.
 *   5. User logs in again — /login now returns mfa_required, the page
 *      swaps to the MFA form, user enters a fresh TOTP code, lands
 *      on the dashboard.
 *   6. User logs out, logs in again, verifies with a backup code.
 *      The same backup code is then rejected on a fresh login
 *      (single-use).
 *   7. User regenerates backup codes; the old ones no longer work.
 *   8. User disables 2FA; subsequent login is password-only.
 */

// fetchAsUser is the in-browser equivalent of curl with the current
// cookie jar. Extracted so each spec can hit the API with the
// identity of whoever most recently logged in on `page`.
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

async function logoutViaButton(page: Page) {
  // Logout link/button lives in the app shell; tolerant matcher
  // because the exact name is i18n'd.
  const logout = page.getByRole('button', { name: /log ?out|sign ?out/i }).first();
  if (await logout.isVisible().catch(() => false)) {
    await logout.click();
  } else {
    // Fallback: hit the logout endpoint directly, then navigate.
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
    await page.goto('/login', { waitUntil: 'load' });
  }
  await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
}

async function enrollTotpViaApi(
  page: Page,
  password: string
): Promise<{ factorId: number; secret: string; backupCodes: string[] }> {
  // Step 1: enrol — returns the factor row + secret + otpauth URI.
  const enrol = await fetchAsUser<{
    id: number;
    enrollment: { secret: string; otpauth_uri: string };
  }>(page, '/api/v1/users/me/factors', {
    method: 'POST',
    body: JSON.stringify({ type: 'totp', password }),
  });
  const secret = extractTotpSecret(enrol.enrollment.otpauth_uri);
  // Step 2: activate with a real TOTP code.
  const code = generateTotp(secret);
  const activate = await fetchAsUser<{
    activated: boolean;
    backup_codes?: { factor_id: number; codes: string[] };
  }>(page, `/api/v1/users/me/factors/${enrol.id}/activate`, {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
  if (!activate.backup_codes) {
    throw new Error('activate did not return backup codes');
  }
  return {
    factorId: enrol.id,
    secret,
    backupCodes: activate.backup_codes.codes,
  };
}

test.describe('Two-factor authentication — full lifecycle', () => {
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'mfa-lifecycle-pw-123';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `mfa-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('MFA Lifecycle User'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    // Log out the admin so the subsequent user login is clean.
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
  });

  test.afterEach(async ({ page }) => {
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {
        /* user may already be removed */
      });
      testUserId = undefined;
    }
  });

  test('enable 2FA from Settings, then login with TOTP, backup code, regenerate, and disable', async ({
    page,
  }) => {
    // ---- Step 1: user logs in with password (no MFA yet) ----
    await loginViaForm(page, testEmail, password);
    // Land on a dashboard-ish page, not on /login.
    await expect(page).not.toHaveURL(/\/login/);

    // ---- Step 2: enable 2FA via Settings UI ----
    await page.goto('/settings', { waitUntil: 'load' });
    await expect(page.getByRole('heading', { name: /two.?factor authentication/i })).toBeVisible({
      timeout: 10000,
    });

    const enableBtn = page.getByRole('button', { name: /enable two.?factor/i });
    await enableBtn.click();

    // Password step of the enrol dialog.
    const dialog = page.getByRole('dialog', { name: /enable two.?factor/i });
    await expect(dialog).toBeVisible();
    await dialog.getByLabel(/current password/i).fill(password);
    await dialog.getByRole('button', { name: /continue/i }).click();

    // QR step. Dialog should contain a QR SVG (from qrcode.react) AND
    // show the fallback secret text.
    await expect(dialog.locator('svg').first()).toBeVisible({ timeout: 10000 });
    const secretNode = dialog.locator('code').first();
    await expect(secretNode).toBeVisible();
    const enrolSecret = (await secretNode.textContent())?.trim() ?? '';
    expect(enrolSecret).toMatch(/^[A-Z2-7]+=*$/i); // base32

    // Generate a real code, enter it in the OTP input, activate.
    const firstCode = generateTotp(enrolSecret);
    // input-otp renders one hidden input with value= the whole code;
    // the easiest way is to target the single <input> inside the
    // dialog by role.
    const otpInput = dialog
      .locator('input')
      .filter({ hasNot: page.locator('[type=password]') })
      .first();
    await otpInput.fill(firstCode);
    await dialog.getByRole('button', { name: /enable two.?factor/i }).click();

    // Backup codes dialog opens. Capture all codes.
    const backupDialog = page.getByTestId('backup-codes-dialog');
    await expect(backupDialog).toBeVisible({ timeout: 10000 });
    const backupCodes = await backupDialog.locator('[data-testid^=backup-code-]').allTextContents();
    expect(backupCodes.length).toBeGreaterThan(4);
    expect(backupCodes.every((c) => c.length > 0)).toBe(true);

    // Acknowledge and close.
    await backupDialog.getByRole('checkbox', { name: /saved these codes/i }).check();
    await backupDialog.getByRole('button', { name: /done/i }).click();
    await expect(backupDialog).not.toBeVisible();

    // Card now shows Enabled badge + action buttons.
    await expect(page.getByRole('button', { name: /disable two.?factor/i })).toBeVisible();

    // ---- Step 3: logout ----
    await logoutViaButton(page);

    // ---- Step 4: login again, MFA gate required ----
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();

    const mfaForm = page.getByTestId('mfa-verify-form');
    await expect(mfaForm).toBeVisible({ timeout: 10000 });

    // Enter a fresh TOTP code.
    const nextCode = generateTotp(enrolSecret);
    await mfaForm.getByLabel(/code/i).fill(nextCode);
    await mfaForm.getByRole('button', { name: /verify/i }).click();

    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // ---- Step 5: logout, login via backup code ----
    await logoutViaButton(page);
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    // Pick backup_codes via the picker (2 factors → picker rendered).
    // Radix Select: open + pick option by text match.
    const picker = page.getByLabel(/verify with/i);
    await picker.click();
    await page.getByRole('option', { name: /recovery codes/i }).click();
    await page.getByLabel(/code/i).fill(backupCodes[0]);
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // ---- Step 6: same backup code cannot be reused ----
    await logoutViaButton(page);
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });
    await page.getByLabel(/verify with/i).click();
    await page.getByRole('option', { name: /recovery codes/i }).click();
    await page.getByLabel(/code/i).fill(backupCodes[0]); // reused!
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.getByRole('alert')).toContainText(/invalid code/i, {
      timeout: 10000,
    });
    // Fall back to TOTP to complete this login (so we can regenerate).
    await page.getByLabel(/verify with/i).click();
    await page.getByRole('option', { name: /authenticator/i }).click();
    // Wait a full step beyond the one the nextCode consumed so replay
    // prevention doesn't reject us.
    await page.waitForTimeout(31000);
    await page.getByLabel(/code/i).fill(generateTotp(enrolSecret));
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // ---- Step 7: regenerate backup codes, old codes stop working ----
    await page.goto('/settings', { waitUntil: 'load' });
    await page.getByRole('button', { name: /regenerate recovery codes/i }).click();
    const regenDialog = page.getByRole('dialog', { name: /regenerate recovery codes/i });
    await regenDialog.getByLabel(/current password/i).fill(password);
    await regenDialog.getByRole('button', { name: /generate new codes/i }).click();

    const newBackupDialog = page.getByTestId('backup-codes-dialog');
    await expect(newBackupDialog).toBeVisible({ timeout: 10000 });
    const freshCodes = await newBackupDialog
      .locator('[data-testid^=backup-code-]')
      .allTextContents();
    expect(freshCodes[0]).not.toEqual(backupCodes[1]);
    await newBackupDialog.getByRole('checkbox', { name: /saved these codes/i }).check();
    await newBackupDialog.getByRole('button', { name: /done/i }).click();

    // Try the OLD (now-invalidated) second backup code.
    await logoutViaButton(page);
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible();
    await page.getByLabel(/verify with/i).click();
    await page.getByRole('option', { name: /recovery codes/i }).click();
    await page.getByLabel(/code/i).fill(backupCodes[1]);
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page.getByRole('alert')).toContainText(/invalid code/i, { timeout: 10000 });

    // One of the fresh codes works.
    await page.getByLabel(/code/i).fill(freshCodes[0]);
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // ---- Step 8: disable 2FA, login is password-only again ----
    await page.goto('/settings', { waitUntil: 'load' });
    await page.getByRole('button', { name: /disable two.?factor/i }).click();
    const disableDialog = page.getByRole('dialog', { name: /disable two.?factor/i });
    await disableDialog.getByLabel(/current password/i).fill(password);
    // Fresh TOTP code (wait a step if needed).
    await page.waitForTimeout(31000);
    await disableDialog
      .getByLabel(/authenticator or recovery code/i)
      .fill(generateTotp(enrolSecret));
    await disableDialog.getByRole('button', { name: /disable two.?factor/i }).click();

    // Card reverts to disabled.
    await expect(page.getByRole('button', { name: /enable two.?factor/i })).toBeVisible({
      timeout: 10000,
    });

    // Logout + login — no MFA prompt this time.
    await logoutViaButton(page);
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
    await expect(page.getByTestId('mfa-verify-form')).not.toBeVisible();
  });
});

test.describe('Two-factor authentication — API-enrolled shortcuts', () => {
  // These specs skip the enrolment UI and set up the factor via API
  // so they can focus on the login-side verify flow.
  let testUserId: number | undefined;
  let testEmail = '';
  const password = 'mfa-short-pw-123';

  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    testEmail = `mfa-short-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: uniqueName('MFA Short User'),
      email: testEmail,
      password,
      active: true,
    });
    testUserId = user.id;
    // Log out admin, log in as the test user, enrol + activate via API.
    await fetchAsUser(page, '/api/v1/logout', { method: 'POST', body: '{}' });
    await loginViaForm(page, testEmail, password);
  });

  test.afterEach(async ({ page }) => {
    if (testUserId) {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await deleteUserViaApi(page, testUserId).catch(() => {
        /* user may already be removed */
      });
      testUserId = undefined;
    }
  });

  test('login page swaps to MFA form when the user has an enrolled factor', async ({ page }) => {
    const { secret } = await enrollTotpViaApi(page, password);

    // Logout then login.
    await logoutViaButton(page);
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();

    // MFA form should be visible.
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible({ timeout: 10000 });

    // Back button reverts to the password form.
    await page.getByRole('button', { name: /^back$/i }).click();
    await expect(page.getByLabel(/email/i)).toBeVisible();

    // Re-enter password, complete with a real code.
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(password);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page.getByTestId('mfa-verify-form')).toBeVisible();
    await page.getByLabel(/code/i).fill(generateTotp(secret));
    await page.getByRole('button', { name: /verify/i }).click();
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  });
});
