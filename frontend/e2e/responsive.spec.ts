import { test, expect } from '@playwright/test';
import { login } from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

test.describe('Responsive Layout - Mobile', () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should display children table with reduced columns on mobile', async ({ page }) => {
    // Navigate to children page
    await page.waitForLoadState('networkidle');

    // Open mobile sidebar to navigate
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });
    await hamburger.click();

    const sidebarOverlay = page.locator('div.fixed.inset-0.z-50');
    await expect(sidebarOverlay).toBeVisible({ timeout: 5000 });

    // Navigate to children via org-scoped nav
    const childrenLink = sidebarOverlay.getByRole('link', { name: /children/i }).first();
    await expect(childrenLink).toBeVisible();
    await childrenLink.click();

    await page.waitForLoadState('networkidle');

    // Name and Actions columns should be visible
    const nameHeader = page.getByRole('columnheader', { name: /name/i });
    await expect(nameHeader.first()).toBeVisible({ timeout: 10000 });

    // Gender column should be hidden on mobile (hidden md:table-cell)
    const genderHeader = page.getByRole('columnheader', { name: /gender/i });
    await expect(genderHeader).not.toBeVisible();
  });

  test('should stack filter bar on mobile', async ({ page }) => {
    // Navigate to children page
    await page.waitForLoadState('networkidle');

    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });
    await hamburger.click();

    const sidebarOverlay = page.locator('div.fixed.inset-0.z-50');
    await expect(sidebarOverlay).toBeVisible({ timeout: 5000 });

    const childrenLink = sidebarOverlay.getByRole('link', { name: /children/i }).first();
    await expect(childrenLink).toBeVisible();
    await childrenLink.click();

    await page.waitForLoadState('networkidle');

    // Filter controls should be visible and wrapped (flex-wrap)
    const filterBar = page.locator('.flex.flex-wrap.items-center').first();
    await expect(filterBar).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Responsive Layout - Desktop', () => {
  test.use({ viewport: { width: 1280, height: 800 } });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should show full table columns on desktop', async ({ page }) => {
    // Navigate to children page via sidebar
    await page.waitForLoadState('networkidle');

    const childrenLink = page.getByRole('link', { name: /children/i }).first();
    await expect(childrenLink).toBeVisible({ timeout: 10000 });
    await childrenLink.click();

    await page.waitForLoadState('networkidle');

    // All columns should be visible on desktop
    const nameHeader = page.getByRole('columnheader', { name: /name/i });
    await expect(nameHeader.first()).toBeVisible({ timeout: 10000 });

    const genderHeader = page.getByRole('columnheader', { name: /gender/i });
    await expect(genderHeader).toBeVisible();
  });

  test('should show desktop sidebar', async ({ page }) => {
    // Desktop sidebar should be visible
    const sidebar = page.locator('aside').first();
    await expect(sidebar).toBeVisible({ timeout: 10000 });

    // Hamburger should not be visible on desktop
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).not.toBeVisible();
  });
});
