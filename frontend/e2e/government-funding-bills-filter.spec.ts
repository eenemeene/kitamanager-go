import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Government Funding Bills - Kita Year Filter', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
  });

  test('should display kita year stepper on bills page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);

    // Verify the page loads with kita year stepper
    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Kita year stepper should be visible with previous/next buttons and year label
    await expect(page.getByRole('button', { name: /previous/i })).toBeVisible();

    // Previous and next buttons should be available
    await expect(page.getByRole('button', { name: /previous/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /next/i })).toBeVisible();
  });

  test('should filter bills by kita year when stepping', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);

    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Get the current kita year label from the stepper
    const yearLabel = page.locator('.min-w-\\[80px\\]').first();
    const initialYear = await yearLabel.textContent();

    // Click previous to go back one year
    await page.getByRole('button', { name: /previous/i }).click();
    await page.waitForTimeout(1000);

    // Year label should have changed
    const newYear = await yearLabel.textContent();
    expect(newYear).not.toBe(initialYear);

    // The card title should include the kita year
    await expect(page.locator('text=/Kita Year/')).toBeVisible();
  });

  test('should show summary bar when bills have comparison data', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);

    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Navigate to a kita year that has bills by stepping backwards if current is empty
    const noBills = page.getByText(/no funding bills uploaded/i);
    let attempts = 0;
    while ((await noBills.count()) > 0 && attempts < 5) {
      await page.getByRole('button', { name: /previous/i }).click();
      await page.waitForTimeout(2000);
      attempts++;
    }

    // If we found bills, verify the summary bar appears
    if (attempts < 5) {
      // Summary badges should appear (match/difference counts)
      await expect(page.getByText(/bills? match|bills? with differences/i).first()).toBeVisible({
        timeout: 10000,
      });

      // Total difference should be shown
      await expect(page.getByText(/total difference/i)).toBeVisible();

      // Table should be visible with rows
      await expect(page.locator('table')).toBeVisible();
    }
  });
});
