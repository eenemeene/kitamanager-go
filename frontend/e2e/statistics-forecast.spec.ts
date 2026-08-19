import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

/**
 * The forecast: model a scenario, calculate, read the impact.
 *
 * statistics.spec.ts asserted the hub lists a Forecast card and then never
 * followed it, so the page itself had no coverage. Its whole purpose is the
 * round trip -- a configuration that produces results -- which is exactly what a
 * "the heading is visible" test would miss.
 */
test.use({ locale: 'en-US' });

test.describe('Forecast', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await getFirstOrganization(page)).id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/statistics/forecast`);
    await page.waitForLoadState('load');
  });

  test('offers the three ways to shape a scenario, and says what to do', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /forecast/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // The strategies are the page's actual interface; losing one silently would
    // leave a user unable to build the scenario they came for.
    for (const tab of [/optimize/i, /add.*remove children/i, /add.*remove employees/i]) {
      await expect(page.getByRole('tab', { name: tab })).toBeVisible({ timeout: 10000 });
    }

    // Before calculating, the page says so rather than showing an empty chart
    // that looks like a forecast of zero.
    await expect(page.getByText(/configure your scenario/i)).toBeVisible();
  });

  test('calculating replaces the prompt with results', async ({ page }) => {
    const prompt = page.getByText(/configure your scenario/i);
    await expect(prompt).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /^calculate forecast$/i }).click();

    // The round trip: a scenario in, an impact out. Both halves matter -- the
    // prompt going away is what tells the user the run happened.
    await expect(page.getByText(/forecast results/i)).toBeVisible({ timeout: 30000 });
    await expect(prompt).toHaveCount(0);
    await expect(page.getByText(/cumulative balance/i).first()).toBeVisible();
  });

  test('states the baseline it is forecasting against', async ({ page }) => {
    // A projection with nothing to compare it to is not actionable; the baseline
    // is what makes the result mean anything.
    await expect(page.getByText(/current baseline/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/current baseline/i)).toContainText(/\d[\d.,]*\s*€/);
  });
});
