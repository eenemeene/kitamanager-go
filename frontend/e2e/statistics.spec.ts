import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Statistics', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
  });

  test('should display statistics hub page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics`);

    // Wait for heading to render (avoid networkidle — react-query background requests prevent it)
    await expect(
      page.getByRole('heading', { name: /statistics/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Verify financial summary cards
    await expect(page.getByText(/total income/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/total expenses/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('heading', { name: /balance/i })).toBeVisible({
      timeout: 10000,
    });

    // Verify navigation cards to sub-pages (rendered as headings within cards)
    await expect(
      page.getByRole('heading', { name: /financials/i })
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { name: /staffing/i })
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { name: /children/i })
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { name: /occupancy/i })
    ).toBeVisible();
  });

  test('should navigate to children statistics', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/children`);

    // Wait for heading (avoid networkidle — react-query background requests prevent it)
    await expect(
      page.getByRole('heading', { name: /children/i }).first()
    ).toBeVisible();

    // Verify chart cards are visible
    await expect(page.getByText(/age distribution/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/contract properties/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to staffing statistics', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/staffing`);

    // Wait for heading (avoid networkidle — react-query background requests prevent it)
    await expect(
      page.getByRole('heading', { name: /staffing/i }).first()
    ).toBeVisible();

    // Verify staffing hours chart card
    await expect(page.getByText(/staffing hours/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to financials statistics', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/financials`);

    // Wait for heading (avoid networkidle — react-query background requests prevent it)
    await expect(
      page.getByRole('heading', { name: /financials/i }).first()
    ).toBeVisible();

    // Verify financial summary cards (use heading role to avoid matching chart SVG legends)
    await expect(page.getByRole('heading', { name: /total income/i })).toBeVisible({
      timeout: 10000,
    });
    await expect(page.getByRole('heading', { name: /total expenses/i })).toBeVisible({
      timeout: 10000,
    });

    // Verify financial overview chart card
    await expect(page.getByText(/financial overview/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to occupancy statistics', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/occupancy`);

    // Wait for heading (avoid networkidle — react-query background requests prevent it)
    await expect(
      page.getByRole('heading', { name: /occupancy/i }).first()
    ).toBeVisible();

    // Verify occupancy matrix card
    await expect(page.getByText(/occupancy matrix/i)).toBeVisible({ timeout: 10000 });
  });
});
