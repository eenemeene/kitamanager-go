import { test, expect } from '@playwright/test';
import { login } from './utils/test-helpers';

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should display dashboard after login', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });

  test('should navigate to organizations page', async ({ page }) => {
    await page.getByRole('link', { name: /organization/i }).first().click();

    await expect(page).toHaveURL(/.*organization/);
    // Use first() since there may be multiple headings with "organization"
    await expect(page.getByRole('heading', { name: /organization/i }).first()).toBeVisible();
  });

  test('should navigate to government fundings page', async ({ page }) => {
    await page.getByRole('link', { name: /government funding|förderung/i }).first().click();

    await expect(page).toHaveURL(/.*government-funding/);
  });

  test('should show sidebar navigation items', async ({ page }) => {
    // Check for main navigation links
    await expect(page.getByRole('link', { name: /organization/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /government funding|förderung/i }).first()).toBeVisible();
  });

  test('should show organization selector', async ({ page }) => {
    // Organization selector should be visible
    const orgSelector = page.locator('button').filter({ hasText: /select|organization|kita/i }).first();
    await expect(orgSelector).toBeVisible({ timeout: 10000 });
  });
});
