import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createBudgetItemViaApi,
  deleteBudgetItemViaApi,
  getBudgetItemsViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Budget Items', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'BudgetItems');
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
    await page.goto(`/organizations/${orgId}/budget-items`);
    await page.waitForLoadState('networkidle');
  });

  test('should display budget items list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /budget item/i }).first()).toBeVisible();
  });

  test('should create a new budget item via UI', async ({ page }) => {
    const itemName = uniqueName('TestBudget');

    await page.getByRole('button', { name: /new budget item/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.getByLabel(/^name$/i).fill(itemName);

    await page.getByRole('dialog').getByRole('combobox').click();
    await page.getByRole('option', { name: /income/i }).click();

    await page.getByLabel(/amount/i).clear();
    await page.getByLabel(/amount/i).fill('100');

    await page.getByRole('button', { name: /save/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(itemName)).toBeVisible({ timeout: 10000 });

    const items = await getBudgetItemsViaApi(page, orgId);
    const created = items.find((i) => i.name === itemName);
    if (created) {
      await deleteBudgetItemViaApi(page, orgId, created.id);
    }
  });

  test('should edit a budget item via UI', async ({ page }) => {
    const origName = uniqueName('EditBudget');
    const item = await createBudgetItemViaApi(page, orgId, {
      name: origName,
      category: 'expense',
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
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

    await deleteBudgetItemViaApi(page, orgId, item.id);
  });

  test('should delete a budget item via UI', async ({ page }) => {
    const itemName = uniqueName('DelBudget');
    await createBudgetItemViaApi(page, orgId, {
      name: itemName,
      category: 'income',
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(itemName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: itemName });
    await row.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText(itemName)).not.toBeVisible({ timeout: 10000 });
  });
});
