/**
 * Screenshot capture script for KitaManager Go website.
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
const OUTPUT_DIR = path.resolve(__dirname, '../static/images/screenshots');

const ADMIN_EMAIL = 'admin@example.com';
const ADMIN_PASSWORD = 'supersecret';

async function login(page: Page): Promise<string> {
  // Navigate to login page to initialize browser context
  await page.goto(`${BASE_URL}/login`);
  await page.waitForLoadState('networkidle');

  // Login via API to get token
  const authData = await page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      if (!response.ok) {
        throw new Error(`Login failed: ${response.status}`);
      }
      return response.json();
    },
    { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
  );

  if (!authData?.token) {
    throw new Error('Failed to obtain auth token');
  }

  // Set cookie
  await page.context().addCookies([
    {
      name: 'token',
      value: authData.token,
      domain: new URL(BASE_URL).hostname,
      path: '/',
      httpOnly: false,
      secure: false,
      sameSite: 'Strict',
    },
  ]);

  // Set localStorage for client-side auth state
  await page.evaluate((token) => {
    localStorage.setItem('token', token);
    localStorage.setItem('auth-storage', JSON.stringify({ state: { token }, version: 0 }));
  }, authData.token);

  return authData.token;
}

async function getFirstOrgId(page: Page, token: string): Promise<number> {
  return page.evaluate(async ({ token }) => {
    const response = await fetch('/api/v1/organizations?limit=1', {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No organizations found — is the database seeded?');
    }
    return data.data[0].id;
  }, { token });
}

async function capture(page: Page, name: string): Promise<void> {
  const filepath = path.join(OUTPUT_DIR, `${name}.png`);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`  ✓ ${name}`);
}

async function main(): Promise<void> {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const browser: Browser = await chromium.launch({ headless: true });
  const context: BrowserContext = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    locale: 'en-US',
  });
  const page: Page = await context.newPage();

  try {
    // 1. Login page (before auth)
    console.log('Capturing screenshots...');
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await capture(page, 'login');

    // 2. Authenticate
    const token = await login(page);

    // 3. Dashboard
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, 'dashboard');

    // 4. Organizations
    await page.goto(`${BASE_URL}/organizations`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, 'organizations');

    // Get first org for scoped pages
    const orgId = await getFirstOrgId(page, token);

    // 5. Employees
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, 'employees');

    // 6. Children
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, 'children');

    // 7. Government Funding
    await page.goto(`${BASE_URL}/government-funding-rates`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await capture(page, 'government-funding-rates');

    console.log(`\nDone! Screenshots saved to ${OUTPUT_DIR}`);
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
