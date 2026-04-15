import { test, expect, Page } from '@playwright/test';
import { login } from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

/** Open the mobile sidebar if the viewport is narrow (hamburger menu visible). */
async function ensureSidebarVisible(page: Page) {
  await page.waitForLoadState('load');
  const hamburger = page.getByRole('button', { name: /menu/i });
  const isHamburgerVisible = await hamburger.isVisible().catch(() => false);
  if (isHamburgerVisible) {
    // Wait for React hydration so the click handler is attached
    await page.waitForTimeout(300);
    await hamburger.click();
    await page.locator('div.fixed.inset-0.z-50').waitFor({ state: 'visible', timeout: 5000 });
  }
}

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should display dashboard after login', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });

  test('should navigate to organizations page', async ({ page }) => {
    await page.waitForLoadState('load');
    await ensureSidebarVisible(page);

    const link = page.getByRole('link', { name: /organization/i }).first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();

    await expect(page).toHaveURL(/\/organizations\/?$/, { timeout: 10000 });
    await expect(page.getByRole('heading', { name: /organization/i }).first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to government fundings page', async ({ page }) => {
    await page.waitForLoadState('load');
    await ensureSidebarVisible(page);

    const link = page.getByRole('link', { name: /government funding/i }).first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();

    await expect(page).toHaveURL(/.*government-funding/, { timeout: 10000 });
  });

  test('should show sidebar navigation items', async ({ page }) => {
    await ensureSidebarVisible(page);
    await expect(page.getByRole('link', { name: /organization/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /government funding/i }).first()).toBeVisible();
  });

  test('should show organization selector', async ({ page }) => {
    await ensureSidebarVisible(page);
    // On mobile there are two org-selectors (desktop hidden + mobile overlay).
    // Use locator that resolves only visible elements.
    const orgSelector = page.locator('[data-testid="org-selector"]:visible').first();
    await expect(orgSelector).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Mobile Navigation', () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should show hamburger menu on mobile', async ({ page }) => {
    // Hamburger menu should be visible
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });

    // Desktop sidebar should not be visible (it has hidden md:flex)
    const sidebar = page.locator('aside').first();
    await expect(sidebar).not.toBeVisible();
  });

  test('should open and close mobile sidebar', async ({ page }) => {
    await page.waitForLoadState('load');

    // Open sidebar via hamburger
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });
    await hamburger.click();

    // Sidebar navigation should appear (use role-based selector)
    const sidebarNav = page.locator('div.fixed.inset-0.z-50 nav');
    await expect(sidebarNav).toBeVisible({ timeout: 5000 });

    // Close by clicking backdrop
    const backdrop = page.locator('div.fixed.inset-0.bg-black\\/50');
    await backdrop.click({ force: true });

    // Sidebar navigation should disappear
    await expect(sidebarNav).not.toBeVisible({ timeout: 5000 });
  });

  test('should navigate via mobile sidebar', async ({ page }) => {
    await page.waitForLoadState('load');

    // Open sidebar
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });
    await hamburger.click();

    // Wait for sidebar navigation to appear
    const sidebarOverlay = page.locator('div.fixed.inset-0.z-50');
    await expect(sidebarOverlay).toBeVisible({ timeout: 5000 });

    // Click on Organizations link in the mobile sidebar
    const orgLink = sidebarOverlay.getByRole('link', { name: /organization/i }).first();
    await expect(orgLink).toBeVisible({ timeout: 5000 });
    await orgLink.click();

    // Should navigate and close sidebar
    await expect(page).toHaveURL(/\/organizations\/?$/, { timeout: 10000 });
    await expect(sidebarOverlay).not.toBeVisible({ timeout: 5000 });
  });
});
