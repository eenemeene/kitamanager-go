import { test, expect } from '@playwright/test';
import { login, getApiToken, getFirstOrganization } from './utils/test-helpers';

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
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('organizations list', async ({ page }) => {
    await page.goto('/organizations');
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveScreenshot('organizations-list.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('employees list', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/employees`);
    await page.waitForLoadState('networkidle');
    // Wait for table to render
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('employees-list.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('children list', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/children`);
    await page.waitForLoadState('networkidle');
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('children-list.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('sections board', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/sections`);
    await page.waitForLoadState('networkidle');
    // Wait for the kanban board or section list to render
    await page.waitForTimeout(1000);

    await expect(page).toHaveScreenshot('sections-board.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('statistics page', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/statistics`);
    await page.waitForLoadState('networkidle');
    // Wait for charts to render (dynamically loaded)
    await page.waitForTimeout(2000);

    await expect(page).toHaveScreenshot('statistics-page.png', {
      maxDiffPixelRatio: 0.02, // Slightly higher tolerance for chart rendering
    });
  });
});

test.describe('Visual Regression - Dialogs', () => {
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
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/employees`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await expect(page).toHaveScreenshot('create-employee-dialog.png', {
      maxDiffPixelRatio: 0.01,
    });
  });

  test('create child dialog', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/children`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new child/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await expect(page).toHaveScreenshot('create-child-dialog.png', {
      maxDiffPixelRatio: 0.01,
    });
  });
});
