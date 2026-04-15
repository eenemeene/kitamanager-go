import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createPayPlanViaApi,
  deletePayPlanViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Pay Plan Detail', () => {
  test.describe.configure({ mode: 'serial' });

  let orgId: number;
  let payPlanId: number;
  let payPlanName: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'PayPlanDetail');
    orgId = testOrg.orgId;
    payPlanName = uniqueName('DetailPlan');
    const plan = await createPayPlanViaApi(page, orgId, payPlanName);
    payPlanId = plan.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deletePayPlanViaApi(page, orgId, payPlanId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  /** Switch to Panels view and wait for the view to actually render */
  async function switchToPanelsView(page: import('@playwright/test').Page) {
    await page.getByRole('button', { name: /panels/i }).click();
    await expect(page.getByRole('button', { name: /add.*period/i })).toBeVisible({
      timeout: 5000,
    });
  }

  test('should display pay plan detail page', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await expect(page.getByRole('heading', { name: payPlanName })).toBeVisible({
      timeout: 10000,
    });

    await expect(page.getByRole('button', { name: /panels/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /table/i })).toBeVisible();
  });

  test('should create a period via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await switchToPanelsView(page);

    await page.getByRole('button', { name: /add.*period/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.locator('#from').fill('2024-01-01');
    await page.locator('#weekly_hours').fill('39');
    await page.locator('#employer_contribution_rate').fill('20');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByRole('button', { name: /add.*entry/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should edit a period via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await switchToPanelsView(page);
    await expect(page.getByRole('button', { name: /add.*entry/i })).toBeVisible({
      timeout: 10000,
    });

    const addEntryBtn = page.getByRole('button', { name: /add.*entry/i });
    const editBtn = addEntryBtn.locator('xpath=following-sibling::button[1]');
    await editBtn.click();

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.locator('#weekly_hours').clear();
    await page.locator('#weekly_hours').fill('38.5');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText(/38\.5h/)).toBeVisible({ timeout: 10000 });
  });

  test('should create an entry within a period via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await switchToPanelsView(page);
    await expect(page.getByRole('button', { name: /add.*entry/i })).toBeVisible({
      timeout: 10000,
    });

    await page.getByRole('button', { name: /add.*entry/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.locator('#grade').fill('S8a');
    await page.locator('#step').fill('3');
    await page.locator('#monthly_amount_euros').fill('3500.00');
    await page.locator('#step_min_years').fill('5');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText('S8a')).toBeVisible({ timeout: 10000 });
  });

  test('should delete an entry via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await switchToPanelsView(page);

    await expect(page.getByText('S8a')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: 'S8a' });
    await row.locator('button').last().click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText('S8a')).not.toBeVisible({ timeout: 10000 });
  });

  test('should delete a period via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await switchToPanelsView(page);
    await expect(page.getByRole('button', { name: /add.*entry/i })).toBeVisible({
      timeout: 10000,
    });

    const addEntryBtn = page.getByRole('button', { name: /add.*entry/i });
    const deleteBtn = addEntryBtn.locator('xpath=following-sibling::button[2]');
    await deleteBtn.click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByRole('button', { name: /add.*entry/i })).not.toBeVisible({
      timeout: 10000,
    });
  });

  test('should toggle between panels and table view', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans/${payPlanId}`);
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /panels/i }).click();
    await expect(page.getByRole('button', { name: /add.*period/i })).toBeVisible({
      timeout: 5000,
    });

    await page.getByRole('button', { name: /table/i }).click();
    await expect(page.getByRole('button', { name: /add.*period/i })).not.toBeVisible({
      timeout: 5000,
    });
  });
});
