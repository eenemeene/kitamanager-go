import { test, expect, Page } from '@playwright/test';
import { login, navigateViaMobileSidebar } from './utils/test-helpers';

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
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 5000 });
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
    await expect(page.getByRole('heading', { name: /organization/i }).first()).toBeVisible({
      timeout: 10000,
    });
  });

  test('should navigate to the funding rates page', async ({ page }) => {
    await page.waitForLoadState('load');
    await ensureSidebarVisible(page);

    const link = page.getByRole('link', { name: /funding rates/i }).first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();

    await expect(page).toHaveURL(/.*government-funding/, { timeout: 10000 });
  });

  test('should show sidebar navigation items', async ({ page }) => {
    await ensureSidebarVisible(page);
    await expect(page.getByRole('link', { name: /organization/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /funding rates/i }).first()).toBeVisible();
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
    const sidebarNav = page.getByRole('dialog').locator('nav');
    await expect(sidebarNav).toBeVisible({ timeout: 5000 });

    // Close by clicking backdrop
    const backdrop = page.locator('div.fixed.inset-0.bg-black\\/50');
    await backdrop.click({ force: true });

    // Sidebar navigation should disappear
    await expect(sidebarNav).not.toBeVisible({ timeout: 5000 });
  });

  test('the drawer survives a redirect the user did not ask for', async ({ page }) => {
    // Landing on `/` renders this whole shell and only then redirects to the org
    // dashboard, once the organization list arrives. A menu opened in that second
    // used to be torn down by the redirect, because the drawer closed reactively
    // on any pathname change rather than on the tap that caused one.
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /menu/i }).click();
    const overlay = page.getByRole('dialog');
    await expect(overlay).toBeVisible({ timeout: 10000 });

    // Wait past the redirect, then require the drawer to still be there.
    await expect(page).toHaveURL(/\/organizations\/\d+\/dashboard$/, { timeout: 15000 });
    await expect(overlay).toBeVisible();
  });

  test('should navigate via mobile sidebar', async ({ page }) => {
    await page.waitForLoadState('load');

    // Opening and clicking are still retried together: React attaches the
    // handler on hydration, so a tap before then is swallowed with no feedback.
    // What no longer needs retrying is the drawer closing underneath -- it now
    // closes on the tap that navigates, not on the path changing afterwards.
    await navigateViaMobileSidebar(page, /organization/i, /\/organizations\/?$/);

    // The tap closes the drawer behind you and still navigates. Both, because
    // closing unmounts the anchor: do it too eagerly and the navigation is lost.
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 });
  });
});
