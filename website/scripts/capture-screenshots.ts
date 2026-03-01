/**
 * Screenshot capture script for KitaManager Go website.
 *
 * Captures screenshots in all supported languages (en, de).
 *
 * Prerequisites:
 *   - API server running on http://localhost:8080 (with seeded data)
 *   - Next.js frontend running on http://localhost:3000
 *
 * Run from the frontend/ directory:
 *   npx tsx ../website/scripts/capture-screenshots.ts
 *
 * Or from the repo root:
 *   cd frontend && npx tsx ../website/scripts/capture-screenshots.ts
 */
import { chromium, type Browser, type Page, type BrowserContext } from 'playwright-core';
import * as path from 'path';
import * as fs from 'fs';

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const OUTPUT_BASE_DIR = path.resolve(__dirname, '../static/images/screenshots');

const ADMIN_EMAIL = 'admin@example.com';
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
  await page.goto(`${BASE_URL}/login`);
  await page.waitForLoadState('networkidle');

  // Login via API — sets HttpOnly access_token and JS-readable csrf_token cookies
  await page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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
    const response = await fetch('/api/v1/organizations?limit=1');
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No organizations found — is the database seeded?');
    }
    return data.data[0].id;
  });
}

async function getFirstEmployeeId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const response = await fetch(`/api/v1/organizations/${orgId}/employees?limit=1`);
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No employees found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstChildId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const response = await fetch(`/api/v1/organizations/${orgId}/children?limit=1`);
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No children found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function getFirstBudgetItemId(page: Page, orgId: number): Promise<number> {
  return page.evaluate(async (orgId) => {
    const response = await fetch(`/api/v1/organizations/${orgId}/budget-items?limit=1`);
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No budget items found — is the database seeded?');
    }
    return data.data[0].id;
  }, orgId);
}

async function capture(page: Page, outputDir: string, name: string): Promise<void> {
  const filepath = path.join(outputDir, `${name}.png`);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`  ✓ ${name}`);
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
    await page.waitForLoadState('networkidle');
    await capture(page, outputDir, 'login');

    // 2. Authenticate and set locale
    await login(page);
    await setLocale(context, lang.code);

    // 3. Dashboard
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'dashboard');

    // 4. Organizations
    await page.goto(`${BASE_URL}/organizations`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'organizations');

    // Get first org for scoped pages
    const orgId = await getFirstOrgId(page);

    // 5. Employees
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'employees');

    // 6. Children
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'children');

    // 7. Government Funding Rates
    await page.goto(`${BASE_URL}/government-funding-rates`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'government-funding-rates');

    // 8. Sections
    await page.goto(`${BASE_URL}/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'sections');

    // 9. Employee Contracts
    const employeeId = await getFirstEmployeeId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees/${employeeId}/contracts`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'employee-contracts');

    // 10. Child Contracts
    const childId = await getFirstChildId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/children/${childId}/contracts`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'child-contracts');

    // 11. Attendance
    await page.goto(`${BASE_URL}/organizations/${orgId}/attendance`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'attendance');

    // 12. Budget Items
    await page.goto(`${BASE_URL}/organizations/${orgId}/budget-items`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'budget-items');

    // 13. Budget Item Detail
    const budgetItemId = await getFirstBudgetItemId(page, orgId);
    await page.goto(`${BASE_URL}/organizations/${orgId}/budget-items/${budgetItemId}`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'budget-item-detail');

    // 14. Statistics Overview
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'statistics');

    // 15. Statistics: Staffing Hours
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/staffing`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await capture(page, outputDir, 'statistics-staffing');

    // 16. Statistics: Financial Overview
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/financials`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await capture(page, outputDir, 'statistics-financials');

    // 17. Statistics: Children (Age Distribution & Contract Properties)
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/children`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await capture(page, outputDir, 'statistics-children');

    // 18. Statistics: Occupancy
    await page.goto(`${BASE_URL}/organizations/${orgId}/statistics/occupancy`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await capture(page, outputDir, 'statistics-occupancy');

    // 19. Employee Contract Creation Dialog
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees/${employeeId}/contracts`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
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
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
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
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, outputDir, 'government-funding-bills');

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
