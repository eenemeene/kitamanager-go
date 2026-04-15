import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createBudgetItemViaApi,
  deleteBudgetItemViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Budget Item Detail', () => {
  test.describe.configure({ mode: 'serial' });

  let orgId: number;
  let budgetItemId: number;
  let budgetItemName: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'BudgetDetail');
    orgId = testOrg.orgId;
    budgetItemName = uniqueName('DetailBudget');
    const item = await createBudgetItemViaApi(page, orgId, {
      name: budgetItemName,
      category: 'income',
    });
    budgetItemId = item.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteBudgetItemViaApi(page, orgId, budgetItemId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test('should display budget item detail page', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/budget-items/${budgetItemId}`);
    await page.waitForLoadState('load');

    await expect(page.getByRole('heading', { name: budgetItemName })).toBeVisible({
      timeout: 10000,
    });

    await expect(page.getByText(/income/i)).toBeVisible();
  });

  test('should create an entry via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/budget-items/${budgetItemId}`);
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /add.*entry/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.locator('#from').fill('2024-01-01');
    await page.locator('#amount_euros').fill('500.00');
    await page.locator('#notes').fill('Test entry notes');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Test entry notes')).toBeVisible({ timeout: 10000 });
  });

  test('should edit an entry via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/budget-items/${budgetItemId}`);
    await page.waitForLoadState('load');

    await expect(page.getByText('Test entry notes')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: 'Test entry notes' });
    await row.locator('button').first().click();

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.locator('#notes').clear();
    await page.locator('#notes').fill('Updated entry notes');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Updated entry notes')).toBeVisible({ timeout: 10000 });
  });

  test('should delete an entry via UI', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/budget-items/${budgetItemId}`);
    await page.waitForLoadState('load');

    await expect(page.getByText('Updated entry notes')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: 'Updated entry notes' });
    await row.locator('button').last().click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText('Updated entry notes')).not.toBeVisible({ timeout: 10000 });
  });
});
