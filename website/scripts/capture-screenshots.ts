/**
 * Screenshot capture script for KitaManager Go website.
 *
 * Captures screenshots in all supported languages (en, de).
 *
 * Prerequisites:
 *   - API server running on http://localhost:8080 (with seeded data)
 *   - Next.js frontend running on http://localhost:3000
 *   - Database seeded with SEED_TEST_DATA=true (use `make dev`)
 *
 * Run from the frontend/ directory:
 *   npx tsx ../website/scripts/capture-screenshots.ts
 *
 * Or from the repo root:
 *   cd frontend && npx tsx ../website/scripts/capture-screenshots.ts
 */
import { chromium, type Browser, type Page, type BrowserContext } from '@playwright/test';
import { TOTP } from 'otpauth';
import * as path from 'path';
import * as fs from 'fs';

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const OUTPUT_BASE_DIR = path.resolve(__dirname, '../static/images/screenshots');

const ADMIN_EMAIL = process.env.SCREENSHOT_EMAIL || 'admin@example.com';
const ADMIN_PASSWORD = 'supersecret';

interface LangConfig {
  code: string;
  browserLocale: string;
  newContractButton: RegExp;
  enableMfaButton: RegExp;
  optimizeButton: RegExp;
  calculateButton: RegExp;
  forecastChildrenTab: RegExp;
  forecastEmployeesTab: RegExp;
  forecastOptimizeTab: RegExp;
  // aria-label of the pencil edit button rendered next to every row.
  // Same string is used by the contracts table, children/employees list,
  // and budget-item entry table.
  editButtonLabel: string;
  // Text on the "Add Entry" button on the budget-item detail page.
  addEntryButton: RegExp;
  // aria-label of the Kita-year stepper's "back" chevron (common.previousYear).
  previousYearLabel: string;
}

const LANGUAGES: LangConfig[] = [
  {
    code: 'en',
    browserLocale: 'en-US',
    newContractButton: /new contract/i,
    enableMfaButton: /enable two-factor authentication/i,
    optimizeButton: /find optimal/i,
    calculateButton: /calculate forecast/i,
    forecastChildrenTab: /^children$/i,
    forecastEmployeesTab: /^employees$/i,
    forecastOptimizeTab: /^optimize$/i,
    editButtonLabel: 'Edit',
    addEntryButton: /add entry/i,
    previousYearLabel: 'Previous year',
  },
  {
    code: 'de',
    browserLocale: 'de-DE',
    newContractButton: /neuer vertrag/i,
    enableMfaButton: /zwei-faktor-authentifizierung aktivieren/i,
    optimizeButton: /optimale kinderzahl/i,
    calculateButton: /prognose berechnen/i,
    forecastChildrenTab: /^kinder$/i,
    forecastEmployeesTab: /^mitarbeiter$/i,
    forecastOptimizeTab: /^optimieren$/i,
    editButtonLabel: 'Bearbeiten',
    addEntryButton: /eintrag hinzufügen/i,
    previousYearLabel: 'Vorheriges Jahr',
  },
];

async function login(page: Page): Promise<void> {
  // Navigate to a page on the right origin first
  await page.goto(`${BASE_URL}/api/v1/health`, { waitUntil: 'load' });

  // Login via API — sets HttpOnly access_token and JS-readable csrf_token cookies
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
    { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
  );
}

async function setLocale(context: BrowserContext, lang: string): Promise<void> {
  const domain = new URL(BASE_URL).hostname;
  await context.addCookies([
    {
      name: 'locale',
      value: lang,
      domain,
      path: '/',
      httpOnly: false,
      secure: false,
      sameSite: 'Lax',
    },
  ]);
}

async function getFirstOrgId(page: Page): Promise<number> {
  return page.evaluate(async () => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch('/api/v1/organizations?limit=1', {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No organizations found — is the database seeded?');
    }
    return data.data[0].id;
  });
}

async function getFirstEmployeeId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch(`/api/v1/organizations/${orgId}/employees?limit=1`, {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No employees found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstChildId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch(`/api/v1/organizations/${orgId}/children?limit=1`, {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No children found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstBudgetItemId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch(`/api/v1/organizations/${orgId}/budget-items?limit=1`, {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No budget items found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstBillId(page: Page, orgId: number): Promise<number | null> {
  return page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch(`/api/v1/organizations/${orgId}/government-funding-bills?limit=1`, {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      return null;
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstPayPlanId(page: Page, orgId: number): Promise<number | null> {
  return page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch(`/api/v1/organizations/${orgId}/pay-plans?limit=1`, {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      return null;
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstFundingId(page: Page): Promise<number | null> {
  return page.evaluate(async () => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = {};
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];
    const response = await fetch('/api/v1/government-fundings?limit=1', {
      credentials: 'same-origin',
      headers,
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      return null;
    }
    return data.data[0].id;
  });
}

async function capture(page: Page, outputDir: string, name: string): Promise<void> {
  const filepath = path.join(outputDir, `${name}.png`);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`  ✓ ${name}`);
}

/**
 * Click the first row's "Edit" pencil button (matched by aria-label
 * which carries the localized "Edit"/"Bearbeiten" string) and capture
 * the resulting dialog. Used for the PersonFormDialog on the children
 * and employees list pages, and for the contract edit dialog on the
 * contract history pages — same button, same selector.
 */
async function captureEditDialog(
  page: Page,
  outputDir: string,
  lang: LangConfig,
  name: string
): Promise<void> {
  const editBtn = page.locator(`button[aria-label="${lang.editButtonLabel}"]`);
  if (!(await editBtn.count())) {
    console.log(`  ! ${name}: no edit button found, skipping`);
    return;
  }
  await editBtn.first().click();
  await page.waitForTimeout(800);
  await capture(page, outputDir, name);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(400);
}

async function waitForContent(page: Page, timeoutMs = 3000): Promise<void> {
  await page.waitForLoadState('load');
  await page.waitForTimeout(timeoutMs);
}

/**
 * Generate a couple of audit-log entries via a create+update+delete
 * round-trip on a throwaway section. The audit log is otherwise empty
 * in seeded environments because the seeder writes directly to the
 * database and bypasses the handler-level audit logging.
 */
async function seedAuditEntries(page: Page, orgId: number): Promise<void> {
  await page.evaluate(async (orgId) => {
    const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];

    const createResp = await fetch(`/api/v1/organizations/${orgId}/sections`, {
      method: 'POST',
      credentials: 'same-origin',
      headers,
      body: JSON.stringify({
        name: 'Demo (auto-removed)',
        // Optional fields — keep minimal so this works regardless of
        // what the section model requires today.
      }),
    });
    if (!createResp.ok) return;
    const created = await createResp.json();
    const id = created.id ?? created.data?.id;
    if (!id) return;

    await fetch(`/api/v1/organizations/${orgId}/sections/${id}`, {
      method: 'PUT',
      credentials: 'same-origin',
      headers,
      body: JSON.stringify({ name: 'Demo (renamed, auto-removed)' }),
    });

    await fetch(`/api/v1/organizations/${orgId}/sections/${id}`, {
      method: 'DELETE',
      credentials: 'same-origin',
      headers,
    });
  }, orgId);
}

/**
 * Forecast: capture each tab plus a results screenshot after clicking
 * "Calculate Forecast" with no modifications (baseline projection).
 */
async function captureForecast(
  page: Page,
  orgId: number,
  outputDir: string,
  lang: LangConfig
): Promise<void> {
  await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/forecast`);
  await waitForContent(page, 3000);

  // Default tab is "Optimize" — already captured as `forecast.png`. We
  // re-capture explicitly so the file name matches the docs.
  await capture(page, outputDir, 'forecast-optimize');

  // Children tab
  const childrenTab = page.getByRole('tab', { name: lang.forecastChildrenTab });
  if (await childrenTab.count()) {
    await childrenTab.first().click();
    await page.waitForTimeout(800);
    await capture(page, outputDir, 'forecast-children');
  }

  // Employees tab
  const employeesTab = page.getByRole('tab', { name: lang.forecastEmployeesTab });
  if (await employeesTab.count()) {
    await employeesTab.first().click();
    await page.waitForTimeout(800);
    await capture(page, outputDir, 'forecast-employees');
  }

  // Switch back to optimize, click Calculate Forecast to populate
  // results panel with the baseline projection.
  const optimizeTab = page.getByRole('tab', { name: lang.forecastOptimizeTab });
  if (await optimizeTab.count()) {
    await optimizeTab.first().click();
    await page.waitForTimeout(500);
  }
  const calcBtn = page.getByRole('button', { name: lang.calculateButton });
  if (await calcBtn.count()) {
    await calcBtn.first().click();
    // Forecast endpoint can take a moment; results render below.
    await page.waitForTimeout(4000);
    // Scroll results into view so the charts are in the viewport.
    await page.evaluate(() => {
      const headings = document.querySelectorAll('h2, h3, [class*="CardTitle"]');
      for (const h of headings) {
        const t = h.textContent ?? '';
        if (/forecast result|prognoseergebnis|results|ergebnis/i.test(t)) {
          h.scrollIntoView({ behavior: 'instant', block: 'start' });
          return;
        }
      }
      // Fallback: scroll halfway down.
      window.scrollTo(0, document.body.scrollHeight / 2);
    });
    await page.waitForTimeout(800);
    await capture(page, outputDir, 'forecast-results');
  }
}

/**
 * Settings page + the 2FA enrolment flow.
 *
 * Captures: settings (full page, MFA disabled), 2FA password step,
 * 2FA QR step, 2FA backup codes dialog, settings with MFA enabled.
 *
 * After the captures, the script disables MFA again so the seeded
 * `admin@example.com` account remains password-only — this matters so
 * subsequent runs of this script (and the dev environment) don't get
 * stuck at the MFA challenge.
 */
async function captureSettingsAndMfa(
  page: Page,
  outputDir: string,
  lang: LangConfig
): Promise<void> {
  await page.goto(`${BASE_URL}/settings`);
  await waitForContent(page, 1500);
  await capture(page, outputDir, 'settings');

  // Click "Enable two-factor authentication" → password prompt step.
  const enableBtn = page.getByRole('button', { name: lang.enableMfaButton });
  if (!(await enableBtn.count())) {
    console.log('  ! enable MFA button not found, skipping 2FA captures');
    return;
  }
  await enableBtn.first().click();
  await page.waitForTimeout(600);
  await capture(page, outputDir, 'settings-2fa-password');

  // Submit password to advance to the QR/scan step.
  const passwordInput = page.locator('input[type="password"]').first();
  if (!(await passwordInput.count())) {
    console.log('  ! password input not found, aborting 2FA captures');
    await page.keyboard.press('Escape');
    return;
  }
  await passwordInput.fill(ADMIN_PASSWORD);
  await page.keyboard.press('Enter');
  await page.waitForTimeout(1500);
  await capture(page, outputDir, 'settings-2fa-scan');

  // The QR code is rendered as an SVG (not a link), so we can't pull
  // the otpauth URI out of an href. The secret is rendered separately
  // as plain text inside a <code> tag underneath the "Can't scan?"
  // hint — that's our hook.
  const secret = await page.evaluate(() => {
    const codes = Array.from(document.querySelectorAll('code'));
    for (const c of codes) {
      const text = (c.textContent ?? '').replace(/\s+/g, '');
      // Base32 secrets are A–Z plus 2–7, typically 16 or 32 chars long.
      if (/^[A-Z2-7]{16,}$/.test(text)) return text;
    }
    return null;
  });
  if (!secret) {
    console.log('  ! TOTP secret not found in dialog, aborting 2FA activation');
    await page.keyboard.press('Escape');
    return;
  }
  const totp = new TOTP({
    issuer: 'KitaManager',
    label: 'admin',
    algorithm: 'SHA1',
    digits: 6,
    period: 30,
    secret,
  });
  const code = totp.generate();

  // shadcn's InputOTP wraps the `input-otp` library, which renders a
  // real <input> with `inputmode="numeric"` underneath the visible
  // slot boxes. Focus it and type — the library distributes characters
  // into the slots automatically.
  const otpFirst = page
    .locator('input#enrol-code, input[autocomplete="one-time-code"], input[inputmode="numeric"]')
    .first();
  if (await otpFirst.count()) {
    await otpFirst.click();
    await otpFirst.fill(code);
  } else {
    console.log('  ! OTP input not found, aborting 2FA activation');
    await page.keyboard.press('Escape');
    return;
  }
  // Submit the activation form. Enter on the OTP input doesn't fire
  // the form submit (the OTP library swallows the keypress), so we
  // click the activate button explicitly. Button text is "Enable
  // two-factor" / "Aktivieren" depending on locale.
  const activateBtn = page
    .getByRole('button')
    .filter({ hasText: /enable two-factor|aktivieren/i });
  if (await activateBtn.count()) {
    await activateBtn.first().click();
  }
  // Wait for the backup-codes dialog to open. The activation request
  // can take a moment under load.
  await page.waitForTimeout(2500);
  await capture(page, outputDir, 'settings-2fa-backup-codes');

  // Tick the "I've saved these codes" checkbox — without it, the
  // Done button stays disabled and Escape is intercepted, so the
  // dialog can't close.
  const ackCheckbox = page
    .getByRole('checkbox')
    .filter({ hasText: /saved these codes|sicher gespeichert/i });
  if (await ackCheckbox.count()) {
    await ackCheckbox.first().click();
  } else {
    // Fallback: click any checkbox in the open dialog.
    const anyCheckbox = page.locator('[role="dialog"] [role="checkbox"]').first();
    if (await anyCheckbox.count()) await anyCheckbox.click();
  }
  const closeBtns = page
    .getByRole('button')
    .filter({ hasText: /^done$|^fertig$/i });
  if (await closeBtns.count()) {
    await closeBtns.first().click();
    await page.waitForTimeout(800);
  } else {
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
  }

  await capture(page, outputDir, 'settings-2fa-enabled');

  // Disable MFA again — leave the seeded account in a clean state for
  // subsequent script runs and for the dev environment.
  await disableMfaForCurrentUser(page, totp);
}

/**
 * Disable all factors for the current user via direct API calls. The
 * delete endpoint requires the account password plus a current TOTP
 * code, so the caller must pass the TOTP instance produced during
 * enrolment.
 */
async function disableMfaForCurrentUser(page: Page, totp: TOTP): Promise<void> {
  // TOTP codes are single-use: the backend records the last-used time step to
  // defeat replays (internal/models/factor.go). Enrolment has just consumed the
  // current step, so a code generated now would be rejected and the account
  // would stay enrolled — which locks every later run of this script out of the
  // password-only login. Wait for the next step before asking for the delete.
  await waitForNextTotpStep(totp);

  const result = await page.evaluate(
    async ({ password, code }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (csrfMatch) headers['X-CSRF-Token'] = csrfMatch[1];

      const factorsResp = await fetch('/api/v1/users/me/factors', {
        credentials: 'same-origin',
        headers,
      });
      if (!factorsResp.ok) return { ok: false, stage: 'list', status: factorsResp.status };
      const data = await factorsResp.json();
      const factors: Array<{ id: number; type: string }> = data.factors ?? [];
      // Delete the TOTP factor first; the service layer also sweeps
      // the backup_codes factor when the last primary is removed.
      const totpFactor = factors.find((f) => f.type === 'totp');
      if (!totpFactor) return { ok: true, stage: 'none-enrolled', status: 200 };
      const del = await fetch(`/api/v1/users/me/factors/${totpFactor.id}`, {
        method: 'DELETE',
        credentials: 'same-origin',
        headers,
        body: JSON.stringify({ password, code }),
      });
      if (!del.ok) {
        return { ok: false, stage: 'delete', status: del.status, body: await del.text() };
      }
      // Confirm the account really is password-only again.
      const after = await fetch('/api/v1/users/me/factors', {
        credentials: 'same-origin',
        headers,
      });
      const remaining = after.ok ? ((await after.json()).factors ?? []).length : -1;
      return { ok: remaining === 0, stage: 'verify', status: del.status, remaining };
    },
    { password: ADMIN_PASSWORD, code: totp.generate() }
  );

  if (!result.ok) {
    // Loud on purpose. A silent failure here leaves MFA enrolled on the seed
    // admin, and the only way back is re-seeding the database.
    throw new Error(
      `Failed to disable MFA for the seed admin (stage=${result.stage}, ` +
        `status=${result.status}${'remaining' in result ? `, remaining=${result.remaining}` : ''}` +
        `${'body' in result ? `, body=${result.body}` : ''}). ` +
        'The account is still enrolled — re-seed with `make dev-fresh` before running again.'
    );
  }
}

/**
 * Block until the TOTP time step advances, so the next generated code has not
 * been used before. Costs up to one period (30s by default) and only runs once
 * per language pass.
 */
async function waitForNextTotpStep(totp: TOTP): Promise<void> {
  const periodMs = (totp.period ?? 30) * 1000;
  const msIntoStep = Date.now() % periodMs;
  const waitMs = periodMs - msIntoStep + 1000;
  console.log(`  … waiting ${Math.round(waitMs / 1000)}s for a fresh TOTP step before disabling MFA`);
  await new Promise((resolve) => setTimeout(resolve, waitMs));
}

async function captureForLanguage(browser: Browser, lang: LangConfig): Promise<void> {
  const outputDir = path.join(OUTPUT_BASE_DIR, lang.code);
  fs.mkdirSync(outputDir, { recursive: true });

  const context: BrowserContext = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    locale: lang.browserLocale,
  });
  const page: Page = await context.newPage();

  try {
    console.log(`\nCapturing screenshots [${lang.code}]...`);

    // 1. Login page (before auth)
    await page.goto(`${BASE_URL}/login`);
    await waitForContent(page, 1000);
    await capture(page, outputDir, 'login');

    // 2. Authenticate and set locale
    await login(page);
    await setLocale(context, lang.code);

    // 3. Dashboard
    await page.goto(`${BASE_URL}/`);
    await waitForContent(page);
    await capture(page, outputDir, 'dashboard');

    // 4. Organizations
    await page.goto(`${BASE_URL}/organizations`);
    await waitForContent(page);
    await capture(page, outputDir, 'organizations');

    // Get first org for scoped pages
    const orgId = await getFirstOrgId(page);

    // 5. Employees
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees`);
    await waitForContent(page);
    await capture(page, outputDir, 'employees');

    // 5b. Employee personal-data edit dialog. Pencil button has the
    // localized "Edit" aria-label. Same PersonFormDialog is shared by
    // employees and children — see #6b for the child variant.
    await captureEditDialog(page, outputDir, lang, 'employee-edit-personal');

    // 6. Children
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await waitForContent(page);
    await capture(page, outputDir, 'children');

    // 6b. Child personal-data edit dialog.
    await captureEditDialog(page, outputDir, lang, 'child-edit-personal');

    // 7. Government Funding Rates
    await page.goto(`${BASE_URL}/government-funding-rates`);
    await waitForContent(page);
    await capture(page, outputDir, 'government-funding-rates');

    // 8. Sections
    await page.goto(`${BASE_URL}/organizations/${orgId}/sections`);
    await waitForContent(page);
    await capture(page, outputDir, 'sections');

    // 9. Employee Contracts
    const employeeId = await getFirstEmployeeId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees/${employeeId}/contracts`);
    await waitForContent(page);
    await capture(page, outputDir, 'employee-contracts');

    // 10. Child Contracts
    const childId = await getFirstChildId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/children/${childId}/contracts`);
    await waitForContent(page);
    await capture(page, outputDir, 'child-contracts');

    // 11. Attendance
    await page.goto(`${BASE_URL}/organizations/${orgId}/attendance`);
    await waitForContent(page);
    await capture(page, outputDir, 'attendance');

    // 12. Budget Items
    await page.goto(`${BASE_URL}/organizations/${orgId}/budget-items`);
    await waitForContent(page);
    await capture(page, outputDir, 'budget-items');

    // 13. Budget Item Detail
    const budgetItemId = await getFirstBudgetItemId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/budget-items/${budgetItemId}`);
    await waitForContent(page);
    await capture(page, outputDir, 'budget-item-detail');

    // 13b. Budget Item — Add Entry dialog.
    const addEntryBtn = page.getByRole('button', { name: lang.addEntryButton });
    if (await addEntryBtn.count()) {
      await addEntryBtn.first().click();
      await page.waitForTimeout(800);
      await capture(page, outputDir, 'budget-item-entry-add');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(400);
    }

    // 14. Statistics Overview
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics`);
    await waitForContent(page);
    await capture(page, outputDir, 'statistics');

    // 15. Statistics: Staffing Hours
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/staffing`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'statistics-staffing');

    // 16. Statistics: Financial Overview
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/financials`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'statistics-financials');

    // 17. Statistics: Children (Age Distribution & Contract Properties)
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/children`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'statistics-children');

    // 18. Statistics: Occupancy
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/occupancy`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'statistics-occupancy');

    // 19. Employee Contract Creation Dialog
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees/${employeeId}/contracts`);
    await waitForContent(page);
    const employeeCreateBtn = page.locator('button', { hasText: lang.newContractButton });
    if (await employeeCreateBtn.isVisible()) {
      await employeeCreateBtn.click();
      await page.waitForTimeout(1000);
      await capture(page, outputDir, 'employee-contract-create');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    }

    // 19b. Employee Contract Edit Dialog — click the first row's pencil.
    await captureEditDialog(page, outputDir, lang, 'employee-contract-edit');

    // 20. Child Contract Creation Dialog
    await page.goto(`${BASE_URL}/organizations/${orgId}/children/${childId}/contracts`);
    await waitForContent(page);
    const childCreateBtn = page.locator('button', { hasText: lang.newContractButton });
    if (await childCreateBtn.isVisible()) {
      await childCreateBtn.click();
      await page.waitForTimeout(1000);
      await capture(page, outputDir, 'child-contract-create');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    }

    // 20b. Child Contract Edit Dialog.
    await captureEditDialog(page, outputDir, lang, 'child-contract-edit');

    // 21. Government Funding Bills
    await page.goto(`${BASE_URL}/organizations/${orgId}/government-funding-bills`);
    await waitForContent(page);
    // The page opens on the *current* Kita year, but the seeder plants bills for
    // the last six calendar months. In the opening months of a Kita year (Aug
    // onwards) those two barely overlap, so the table would be empty and the
    // screenshot would show nothing useful. Step back until rows appear, so the
    // capture shows real data whichever month it runs in.
    for (let back = 0; back < 2; back++) {
      if ((await page.locator('table tbody tr').count()) > 0) break;
      await page.getByRole('button', { name: lang.previousYearLabel }).click();
      await waitForContent(page);
    }
    await capture(page, outputDir, 'government-funding-bills');

    // 22. Government Funding Bill Detail (if bills exist)
    const billId = await getFirstBillId(page, orgId);
    if (billId) {
      await page.goto(`${BASE_URL}/organizations/${orgId}/government-funding-bills/${billId}`);
      await waitForContent(page, 4000);
      await capture(page, outputDir, 'government-funding-bill-detail');
    }

    // 23. Forecast
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/forecast`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'forecast');

    // 24. Financials: Cumulative Balance (scroll down)
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/financials`);
    await waitForContent(page, 4000);
    await page.evaluate(() => {
      const headings = document.querySelectorAll('h3, div[class*="CardTitle"]');
      for (const h of headings) {
        if (h.textContent?.includes('Cumulative') || h.textContent?.includes('Kumuliert')) {
          h.scrollIntoView({ behavior: 'instant', block: 'start' });
          break;
        }
      }
    });
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'statistics-cumulative-balance');

    // 25. Financials: Actual vs Calculated Funding (scroll)
    await page.evaluate(() => {
      const headings = document.querySelectorAll('h3, div[class*="CardTitle"]');
      for (const h of headings) {
        if (h.textContent?.includes('Actual') || h.textContent?.includes('Soll-Ist')) {
          h.scrollIntoView({ behavior: 'instant', block: 'start' });
          break;
        }
      }
    });
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'statistics-funding-comparison');

    // 26. Financials: Budget Overview (scroll)
    await page.evaluate(() => {
      const headings = document.querySelectorAll('h3, div[class*="CardTitle"]');
      for (const h of headings) {
        if (h.textContent?.includes('Budget')) {
          h.scrollIntoView({ behavior: 'instant', block: 'start' });
          break;
        }
      }
    });
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'statistics-budget');

    // 27. Government Funding Rate Detail (navigate to page, ID 1 is the Berlin config)
    await page.goto(`${BASE_URL}/government-funding-rates/1`);
    await waitForContent(page);
    await capture(page, outputDir, 'government-funding-rate-detail');

    // 28. Pay Plan List + Detail
    await page.goto(`${BASE_URL}/organizations/${orgId}/payplans`);
    await waitForContent(page);
    await capture(page, outputDir, 'payplan-list');
    // Click first view link in the pay plans table to drill into the detail.
    const payPlanLink = page.locator('table a, table button').first();
    if (await payPlanLink.count() > 0) {
      await payPlanLink.click();
      await waitForContent(page);
      await capture(page, outputDir, 'payplan-detail');
    }

    // 29. Child Billing History
    await page.goto(`${BASE_URL}/organizations/${orgId}/children/${childId}/billing`);
    await waitForContent(page, 4000);
    await capture(page, outputDir, 'child-billing');

    // 30. Audit log (admins only — `admin@example.com` has the global admin role on
    // the seeded org, so this works without superadmin). The seed data
    // doesn't include audit entries, so we generate a couple via a
    // create+delete round-trip on a throwaway section. That keeps the
    // dev DB clean while showing realistic rows in the screenshot.
    await seedAuditEntries(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/audit-logs`);
    await waitForContent(page, 1500);
    await capture(page, outputDir, 'audit-logs');

    // 31. Forecast tabs. The default tab is "Optimize"; we capture each
    // tab plus a results capture after running the optimizer.
    await captureForecast(page, orgId, outputDir, lang);

    // 32. Settings + 2FA. Done LAST because enrolling MFA puts the
    // account into an MFA-required state for subsequent logins, which
    // would break a re-run of this script. We disable MFA at the end.
    await captureSettingsAndMfa(page, outputDir, lang);

    console.log(`  Done [${lang.code}]!`);
  } finally {
    await context.close();
  }
}

async function main(): Promise<void> {
  const browser: Browser = await chromium.launch({ headless: true });

  try {
    for (const lang of LANGUAGES) {
      await captureForLanguage(browser, lang);
    }
    console.log(`\nAll screenshots saved to ${OUTPUT_BASE_DIR}`);
  } catch (error) {
    console.error('Error capturing screenshots:', error);
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
