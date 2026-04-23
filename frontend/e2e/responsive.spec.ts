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
    await page.waitForLoadState('load');

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

    await page.waitForLoadState('load');

    // Name and Actions columns should be visible
    const nameHeader = page.getByRole('columnheader', { name: /name/i });
    await expect(nameHeader.first()).toBeVisible({ timeout: 10000 });

    // Gender column should be hidden on mobile (hidden lg:table-cell)
    const genderHeader = page.getByRole('columnheader', { name: /gender/i });
    await expect(genderHeader).not.toBeVisible();
  });

  test('should stack filter bar on mobile', async ({ page }) => {
    // Navigate to children page
    await page.waitForLoadState('load');

    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible({ timeout: 10000 });
    await hamburger.click();

    const sidebarOverlay = page.locator('div.fixed.inset-0.z-50');
    await expect(sidebarOverlay).toBeVisible({ timeout: 5000 });

    const childrenLink = sidebarOverlay.getByRole('link', { name: /children/i }).first();
    await expect(childrenLink).toBeVisible();
    await childrenLink.click();

    await page.waitForLoadState('load');

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
    await page.waitForLoadState('load');

    const childrenLink = page.getByRole('link', { name: /children/i }).first();
    await expect(childrenLink).toBeVisible({ timeout: 10000 });
    await childrenLink.click();

    await page.waitForLoadState('load');

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

/**
 * Tablet portrait coverage. md: breakpoint (768-1023px) catches iPad 10th/11th
 * gen (820px), iPad Pro 11" (834px), and similar. iPad Air/Pro 13" (1024px)
 * hits lg:. The sidebar must render collapsed (icon-only) in this range even
 * if the user's stored preference is expanded, because the content area is
 * only ~700-960px wide and can't afford to lose 256px to a labeled sidebar.
 */
test.describe('Responsive Layout - Tablet', () => {
  test.use({ viewport: { width: 820, height: 1180 }, hasTouch: true });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('sidebar is icon-only at tablet portrait width', async ({ page }) => {
    await page.waitForLoadState('load');

    const sidebar = page.locator('aside').first();
    await expect(sidebar).toBeVisible({ timeout: 10000 });

    // Sidebar rail should be ~64px wide (w-16) at md, not 256px (w-64)
    const box = await sidebar.boundingBox();
    expect(box?.width).toBeLessThanOrEqual(80);

    // Labels (e.g. "Dashboard") should not be rendered as text
    // The nav link exists (by aria-label) but no visible text span
    const dashboardLink = page.getByRole('link', { name: /dashboard/i }).first();
    await expect(dashboardLink).toBeVisible({ timeout: 10000 });

    // Version footer is expanded-only, must be hidden at md
    const versionText = page.getByText(/^version:/i);
    await expect(versionText).not.toBeVisible();
  });

  test('children table keeps action icons on a single row', async ({ page }) => {
    await page.waitForLoadState('load');

    const childrenLink = page.getByRole('link', { name: /children/i }).first();
    await childrenLink.click();
    await page.waitForLoadState('load');

    // Wait for at least one row to render
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    const deleteBtn = page.getByRole('button', { name: /delete/i }).first();
    await expect(deleteBtn).toBeVisible();

    // They should be horizontally aligned, not stacked. Compare Y centers.
    const eb = await editBtn.boundingBox();
    const db = await deleteBtn.boundingBox();
    const editCenterY = (eb?.y ?? 0) + (eb?.height ?? 0) / 2;
    const deleteCenterY = (db?.y ?? 0) + (db?.height ?? 0) / 2;
    expect(Math.abs(editCenterY - deleteCenterY)).toBeLessThan(8);
  });

  test('primary touch targets meet the 44px minimum', async ({ page }) => {
    await page.waitForLoadState('load');

    // Header icon buttons (theme toggle)
    const themeBtn = page.getByRole('button', { name: /(dark mode|light mode|theme)/i }).first();
    await expect(themeBtn).toBeVisible({ timeout: 10000 });
    const themeBox = await themeBtn.boundingBox();
    expect(themeBox?.width).toBeGreaterThanOrEqual(44);
    expect(themeBox?.height).toBeGreaterThanOrEqual(44);

    // Navigate to attendance — the daily-use surface for teachers
    const attendanceLink = page.getByRole('link', { name: /attendance/i }).first();
    await attendanceLink.click();
    await page.waitForLoadState('load');

    // Week/day stepper chevron (explicit aria-label to avoid matching
    // pagination or other "next" controls). The attendance view toggles
    // between day and week modes — accept either.
    const stepperNext = page.getByRole('button', { name: /^next (week|day)$/i }).first();
    await expect(stepperNext).toBeVisible({ timeout: 15000 });
    const nextBox = await stepperNext.boundingBox();
    expect(nextBox?.width).toBeGreaterThanOrEqual(44);
    expect(nextBox?.height).toBeGreaterThanOrEqual(44);
  });
});
