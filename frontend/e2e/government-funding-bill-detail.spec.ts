import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

/**
 * One ISBJ bill, child by child, against what KitaManager calculated.
 *
 * The list page was covered; this detail view -- where a Kita finds out *which*
 * child a discrepancy came from -- was not. The per-child comparison is the
 * feature, so asserting only that the page loaded would miss the thing it exists
 * to do.
 *
 * The bill is taken from the API rather than hard-coded, so this does not depend
 * on which fixture happens to be first.
 */
test.use({ locale: 'en-US' });

test.describe('Government funding bill detail', () => {
  let orgId: number;
  let billId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await getFirstOrganization(page)).id;
    const bills = await page.evaluate(async (o) => {
      const r = await fetch(`/api/v1/organizations/${o}/government-funding-bills?limit=5`, {
        credentials: 'same-origin',
      });
      return r.ok ? await r.json() : { data: [] };
    }, orgId);
    billId = bills.data?.[0]?.id;
    await page.close();
    expect(billId, 'seed data should include at least one imported bill').toBeTruthy();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/government-funding-bills/${billId}`);
    await page.waitForLoadState('load');
  });

  test('states the totals and the gap between them', async ({ page }) => {
    // Two figures and their difference: the reason anyone opens this page.
    await expect(page.getByText(/facility total/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/difference/i).first()).toBeVisible();
    await expect(page.locator('body')).toContainText(/\d[\d.,]*\s*€/);
  });

  test('breaks the bill down per child, with a verdict on each', async ({ page }) => {
    // A total alone cannot be acted on. The comparison names how many children
    // matched and lists the ones that did not.
    await expect(page.getByText(/comparison/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/^match$/i).first()).toBeVisible();
    await expect(page.getByText(/\d+ children/i).first()).toBeVisible();
  });

  test('is reachable from the bills list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/government-funding-bills`);
    await page.waitForLoadState('load');

    // The way in is an icon-only link in the row. It had no accessible name
    // until this branch, so a screen reader announced "link" and nothing else --
    // asserting by name is what keeps that from silently regressing.
    // The list is filtered by Kita year and opens on the current one, which may
    // hold no bills. Step back until a row appears, the same way
    // government-funding-bills-filter.spec.ts does.
    const rows = page.locator('table tbody tr');
    for (let year = 0; year < 4; year += 1) {
      // Poll rather than sleep: the row count settles when the query for that
      // year resolves, and a fixed wait either stalls the run or steps straight
      // past the year that holds the bills.
      const found = await expect
        .poll(() => rows.count(), { timeout: 2000 })
        .toBeGreaterThan(0)
        .then(() => true)
        .catch(() => false);
      if (found) break;
      await page.getByRole('button', { name: /previous year/i }).click();
    }

    const row = rows.first();
    await expect(row, 'seed data should have a bill in one of the last few Kita years').toBeVisible(
      {
        timeout: 10000,
      }
    );
    await row.getByRole('link', { name: /view details/i }).click();

    await expect(page).toHaveURL(/\/government-funding-bills\/\d+$/, { timeout: 10000 });
    await expect(page.getByText(/comparison/i).first()).toBeVisible({ timeout: 10000 });
  });
});
