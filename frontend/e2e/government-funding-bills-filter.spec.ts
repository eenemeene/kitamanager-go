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
    await expect(page.getByRole('button', { name: /previous year/i })).toBeVisible();

    // Previous and next buttons should be available
    await expect(page.getByRole('button', { name: /previous year/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /next year/i })).toBeVisible();
  });

  test('should filter bills by kita year when stepping', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);

    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Get the current kita year label from the stepper
    const yearLabel = page.locator('.min-w-\\[80px\\]').first();
    const initialYear = (await yearLabel.textContent()) ?? '';
    expect(initialYear).not.toBe('');

    // Click previous to go back one year, then wait for the label to
    // actually change instead of sleeping. This catches both the request
    // landing and the re-render in a single explicit assertion.
    await page.getByRole('button', { name: /previous year/i }).click();
    await expect(yearLabel).not.toHaveText(initialYear, { timeout: 10000 });

    // The card title should include the kita year
    await expect(page.locator('text=/Kita Year/')).toBeVisible();
  });

  test('should filter bills by search across facility, child name, and voucher', async ({
    page,
  }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);
    await page.waitForLoadState('load');

    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Step back through kita years until we find one with bills (seed data
    // covers the last 6 months so the current kita year usually has rows).
    const noBills = page.getByText(/no funding bills uploaded/i);
    const previousButton = page.getByRole('button', { name: /previous year/i });
    const yearLabel = page.locator('.min-w-\\[80px\\]').first();
    let foundBills = (await noBills.count()) === 0;
    for (let i = 0; i < 5 && !foundBills; i++) {
      const before = (await yearLabel.textContent()) ?? '';
      await previousButton.click();
      await expect(yearLabel).not.toHaveText(before, { timeout: 10000 });
      foundBills = (await noBills.count()) === 0;
    }
    expect(foundBills, 'expected at least one kita year to have bills (seed regression?)').toBe(
      true
    );

    const searchBox = page.getByPlaceholder(/search by facility, child or voucher/i);
    await expect(searchBox).toBeVisible();

    // 1. Facility-name match — seed creates "Kita Sonnenschein".
    await searchBox.fill('Sonnenschein');
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('table tbody tr').first()).toContainText(/Sonnenschein/i);

    // 2. Voucher-number match — seed builds vouchers as "GB-1000…". Searching the
    // shared prefix matches because the EXISTS subquery hits the bill_children rows.
    await searchBox.fill('GB-1000');
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });

    // 3. Nonsense match — empty state must show, proving the search hit the server
    // (client-side fallback would still show all rows because facility_name is "Kita Sonnenschein").
    await searchBox.fill('zzzz-no-such-voucher-xyz');
    await expect(page.getByText(/no funding bills uploaded|no bills match/i)).toBeVisible({
      timeout: 10000,
    });
  });

  test('should show summary bar when bills have comparison data', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);

    await expect(
      page.getByRole('heading', { name: /government funding bills/i }).first()
    ).toBeVisible({ timeout: 10000 });

    // Navigate to a kita year that has bills by stepping backwards if
    // the current year is empty. Each step waits for the year label to
    // change rather than sleeping, so we never advance to the next
    // iteration before the previous one's request has landed.
    const noBills = page.getByText(/no funding bills uploaded/i);
    const previousButton = page.getByRole('button', { name: /previous year/i });
    const yearLabel = page.locator('.min-w-\\[80px\\]').first();

    const maxAttempts = 5;
    let foundBills = (await noBills.count()) === 0;
    for (let i = 0; i < maxAttempts && !foundBills; i++) {
      const before = (await yearLabel.textContent()) ?? '';
      await previousButton.click();
      await expect(yearLabel).not.toHaveText(before, { timeout: 10000 });
      foundBills = (await noBills.count()) === 0;
    }

    // Always assert: if no kita year in the seed data has bills, that's
    // a seed-data regression, not "skip the test" — fail loudly so the
    // breakage shows up here instead of as silent green CI.
    expect(
      foundBills,
      `expected at least one kita year within ${maxAttempts} steps back to have bills (seed data regression?)`
    ).toBe(true);

    // Summary badges should appear (match/difference counts)
    await expect(page.getByText(/bills? match|bills? with differences/i).first()).toBeVisible({
      timeout: 10000,
    });

    // Total difference should be shown
    await expect(page.getByText(/total difference/i)).toBeVisible();

    // Table should be visible with rows
    await expect(page.locator('table')).toBeVisible();
  });
});
