import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

// Ensure English locale for consistent text rendering
test.use({ locale: 'en-US' });

// Visual regression tests capture baseline screenshots on first run.
// Subsequent runs compare against baselines to detect unintended visual changes.
// Update baselines: npx playwright test visual-regression --update-snapshots

test.describe('Visual Regression - Login', () => {
  test('login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveScreenshot('login-page.png', {
      maxDiffPixelRatio: 0.01,
    });
  });
});

test.describe('Visual Regression - Dashboard', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('organizations list', async ({ page }) => {
    await page.goto('/organizations');
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveScreenshot('organizations-list.png', {
      maxDiffPixelRatio: 0.02,
    });
  });

  test('employees list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');
    // Wait for table to render
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('employees-list.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('children list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('children-list.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('sections board', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');
    // Wait for the kanban board to render
    await expect(page.getByText(/drag children/i)).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('sections-board.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('statistics overview page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics`);
    await page.waitForLoadState('networkidle');
    // Wait for statistics cards to render
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('statistics-overview.png', {
      maxDiffPixelRatio: 0.02,
    });
  });

  test('statistics financials page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/financials`);
    await page.waitForLoadState('networkidle');
    // Wait for the financial overview chart card to render
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // Mask the chart area: SVG rendering has sub-pixel anti-aliasing jitter
    // between runs, and the "Today" marker shifts position over time.
    const chartAreas = page.locator('[role="application"]');
    await expect(page).toHaveScreenshot('statistics-financials.png', {
      maxDiffPixelRatio: 0.01,
      mask: [chartAreas],
    });
  });
});

test.describe('Visual Regression - Dialogs', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('create organization dialog', async ({ page }) => {
    await page.goto('/organizations');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new organization/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await expect(page).toHaveScreenshot('create-organization-dialog.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('create employee dialog', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await expect(page).toHaveScreenshot('create-employee-dialog.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('create child dialog', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new child/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5000 });

    // Screenshot only the dialog element: the children page behind the
    // dialog overlay shows today's date in the week navigator, which
    // changes daily and causes pixel diffs. Allow slightly higher diff
    // ratio for tablet where funding property tags vary by test data state.
    await expect(dialog).toHaveScreenshot('create-child-dialog.png', {
      maxDiffPixelRatio: 0.02,
    });
  });
});
