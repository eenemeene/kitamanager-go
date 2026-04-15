import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createPayPlanViaApi,
  deletePayPlanViaApi,
  getPayPlansViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Pay Plans', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'PayPlans');
    orgId = testOrg.orgId;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/payplans`);
    await page.waitForLoadState('load');
  });

  test('should display pay plans list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /pay plan/i }).first()).toBeVisible();
  });

  test('should create a new pay plan via UI', async ({ page }) => {
    const planName = uniqueName('TestPlan');

    await page.getByRole('button', { name: /new pay plan/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.getByLabel(/name/i).fill(planName);
    await page.getByRole('button', { name: /save/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(planName)).toBeVisible({ timeout: 10000 });

    const plans = await getPayPlansViaApi(page, orgId);
    const created = plans.find((p) => p.name === planName);
    if (created) {
      await deletePayPlanViaApi(page, orgId, created.id);
    }
  });

  test('should edit a pay plan via UI', async ({ page }) => {
    const origName = uniqueName('EditPlan');
    const plan = await createPayPlanViaApi(page, orgId, origName);

    await page.reload();
    await page.waitForLoadState('load');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: origName });
    await row.getByRole('button', { name: /edit/i }).click();

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    const updatedName = uniqueName('Updated');
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(updatedName);

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 });

    await deletePayPlanViaApi(page, orgId, plan.id);
  });

  test('should delete a pay plan via UI', async ({ page }) => {
    const planName = uniqueName('DelPlan');
    await createPayPlanViaApi(page, orgId, planName);

    await page.reload();
    await page.waitForLoadState('load');
    await expect(page.getByText(planName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: planName });
    await row.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText(planName)).not.toBeVisible({ timeout: 10000 });
  });
});
