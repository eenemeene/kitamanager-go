/**
 * Screenshot capture script for KitaManager Go website
 * Run with: npx playwright test scripts/capture-screenshots.ts
 * or: npx tsx scripts/capture-screenshots.ts
 */
import { chromium, Browser, Page } from 'playwright';
import * as path from 'path';
import * as fs from 'fs';

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const OUTPUT_DIR = path.join(__dirname, '../static/images/screenshots');

async function login(page: Page): Promise<void> {
  await page.goto(`${BASE_URL}/login`);
  await page.waitForLoadState('networkidle');

  await page.getByLabel(/email/i).fill('admin@example.com');
  await page.getByLabel(/password/i).fill('admin123');
  await page.getByRole('button', { name: /sign in|login|anmelden/i }).click();

  // Wait for redirect to dashboard
  await page.waitForURL(/.*(?!login)/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

async function captureScreenshot(
  page: Page,
  name: string,
  description: string
): Promise<void> {
  const filepath = path.join(OUTPUT_DIR, `${name}.png`);
  await page.screenshot({
    path: filepath,
    fullPage: false,
  });
  console.log(`Captured: ${name} - ${description}`);
}

async function main(): Promise<void> {
  // Ensure output directory exists
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const browser: Browser = await chromium.launch({
    headless: true,
  });

  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    locale: 'en-US',
  });

  const page: Page = await context.newPage();

  try {
    // 1. Login page
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await captureScreenshot(page, 'login', 'Login page');

    // 2. Login and go to dashboard
    await login(page);
    await captureScreenshot(page, 'dashboard', 'Dashboard overview');

    // 3. Organizations list
    await page.goto(`${BASE_URL}/organizations`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000); // Allow table to render
    await captureScreenshot(page, 'organizations', 'Organizations list');

    // 4. Get first organization for org-scoped pages
    const orgId = await page.evaluate(async () => {
      const token = localStorage.getItem('token');
      if (!token) return 1;
      const res = await fetch('/api/v1/organizations?limit=1', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      return data.data?.[0]?.id || 1;
    });

    // 5. Employees list
    await page.goto(`${BASE_URL}/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await captureScreenshot(page, 'employees', 'Employees list');

    // 6. Children list
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await captureScreenshot(page, 'children', 'Children list with funding');

    // 7. Government funding
    await page.goto(`${BASE_URL}/government-funding`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await captureScreenshot(page, 'government-funding', 'Government funding configuration');

    // 8. Sections
    await page.goto(`${BASE_URL}/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    await captureScreenshot(page, 'sections', 'Organizational sections');

    // 9. New organization dialog
    await page.goto(`${BASE_URL}/organizations`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /new organization/i }).click();
    await page.waitForTimeout(500);
    await captureScreenshot(page, 'new-organization-dialog', 'Create organization dialog');
    await page.keyboard.press('Escape');

    // 10. Child detail with contracts (if we can find one)
    await page.goto(`${BASE_URL}/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Try to click on first child row
    const firstChildRow = page.locator('tbody tr').first();
    if (await firstChildRow.isVisible({ timeout: 5000 }).catch(() => false)) {
      // Look for a view/details link
      const viewButton = firstChildRow.getByRole('link').first();
      if (await viewButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await viewButton.click();
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(1000);
        await captureScreenshot(page, 'child-detail', 'Child detail page');
      }
    }

    console.log('\nAll screenshots captured successfully!');
    console.log(`Output directory: ${OUTPUT_DIR}`);
  } catch (error) {
    console.error('Error capturing screenshots:', error);
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch(console.error);
