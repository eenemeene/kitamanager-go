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
}

const LANGUAGES: LangConfig[] = [
  { code: 'en', browserLocale: 'en-US', newContractButton: /new contract/i },
  { code: 'de', browserLocale: 'de-DE', newContractButton: /neuer vertrag/i },
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

async function waitForContent(page: Page, timeoutMs = 3000): Promise<void> {
  await page.waitForLoadState('load');
  await page.waitForTimeout(timeoutMs);
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

    // 6. Children
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await waitForContent(page);
    await capture(page, outputDir, 'children');

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

    // 21. Government Funding Bills
    await page.goto(`${BASE_URL}/organizations/${orgId}/government-funding-bills`);
    await waitForContent(page);
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

    // 28. Pay Plan Detail (navigate to first one)
    await page.goto(`${BASE_URL}/organizations/${orgId}/payplans`);
    await waitForContent(page);
    // Click first view link in the pay plans table
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
