import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Statistics Print Pages', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
  });

  test('staffing print page renders without sidebar', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/staffing/print`);
    await page.waitForLoadState('networkidle');

    // Verify heading
    await expect(page.getByRole('heading', { name: /staffing/i }).first()).toBeVisible();

    // Verify print button is visible
    await expect(page.getByRole('button', { name: /print/i })).toBeVisible();

    // Verify no sidebar
    await expect(page.locator('[class*="sidebar"]')).not.toBeVisible();

    // Verify staffing content loads
    await expect(page.getByText(/staffing hours/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('financials print page renders without sidebar', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/financials/print`);

    await expect(page.getByRole('heading', { name: /financials/i }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /print/i })).toBeVisible();
    await expect(page.locator('[class*="sidebar"]')).not.toBeVisible();

    // Verify financial summary cards
    await expect(page.getByText(/total income/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/total expenses/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('occupancy print page renders without sidebar', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/occupancy/print`);

    await expect(page.getByRole('heading', { name: /occupancy/i }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /print/i })).toBeVisible();
    await expect(page.locator('[class*="sidebar"]')).not.toBeVisible();

    // Verify occupancy content loads
    await expect(page.getByText(/occupancy matrix/i)).toBeVisible({ timeout: 10000 });
  });

  test('children print page renders without sidebar', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/children/print`);

    await expect(page.getByRole('heading', { name: /children/i }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /print/i })).toBeVisible();
    await expect(page.locator('[class*="sidebar"]')).not.toBeVisible();

    // Verify children content loads
    await expect(page.getByText(/age distribution/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('print page redirects to login when unauthenticated', async ({ page }) => {
    // Clear cookies to simulate unauthenticated state
    await page.context().clearCookies();
    await page.goto(`/organizations/${orgId}/statistics/staffing/print`);

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });

  test('dashboard staffing page has print link', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/staffing`);

    const printLink = page.getByRole('link', { name: /print/i });
    await expect(printLink).toBeVisible();
    await expect(printLink).toHaveAttribute('target', '_blank');
    await expect(printLink).toHaveAttribute(
      'href',
      `/organizations/${orgId}/statistics/staffing/print`
    );
  });
});
