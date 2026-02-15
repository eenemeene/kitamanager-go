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
    // Wait for initial redirect after login to complete
    await page.waitForLoadState('networkidle');

    const link = page.getByRole('link', { name: /organization/i }).first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();

    await expect(page).toHaveURL(/\/organizations\/?$/, { timeout: 10000 });
    // Use first() since there may be multiple headings with "organization"
    await expect(page.getByRole('heading', { name: /organization/i }).first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to government fundings page', async ({ page }) => {
    // Wait for initial redirect after login to complete
    await page.waitForLoadState('networkidle');

    const link = page.getByRole('link', { name: /government funding|förderung/i }).first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();

    await expect(page).toHaveURL(/.*government-funding/, { timeout: 10000 });
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
