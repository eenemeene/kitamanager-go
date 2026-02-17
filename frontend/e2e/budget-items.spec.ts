import { test, expect } from '@playwright/test';
import {
  login,
  getFirstOrganization,
  createBudgetItemViaApi,
  deleteBudgetItemViaApi,
  getBudgetItemsViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Budget Items', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.goto(`/organizations/${orgId}/budget-items`);
    await page.waitForLoadState('networkidle');
  });

  test('should display budget items list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /budget item/i }).first()).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new budget item via UI', async ({ page }) => {
    const itemName = uniqueName('TestBudget');

    // Click "New Budget Item" button
    await page.getByRole('button', { name: /new budget item/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form fields
    await page.getByLabel(/name/i).fill(itemName);

    // Select category
    await page.getByRole('dialog').getByRole('combobox').click();
    await page.getByRole('option', { name: /income/i }).click();

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Budget item should appear in list
    await expect(page.getByText(itemName)).toBeVisible({ timeout: 10000 });

    // Cleanup via API
    const items = await getBudgetItemsViaApi(page, orgId);
    const created = items.find((i) => i.name === itemName);
    if (created) {
      await deleteBudgetItemViaApi(page, orgId, created.id);
    }
  });

  test('should edit a budget item via UI', async ({ page }) => {
    // Setup: create budget item via API
    const origName = uniqueName('EditBudget');
    const item = await createBudgetItemViaApi(page, orgId, {
      name: origName,
      category: 'expense',
    });

    // Reload to see the item
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    // Click edit button on the item's row
    const row = page.getByRole('row').filter({ hasText: origName });
    await row.getByRole('button', { name: /edit/i }).click();

    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Modify name
    const updatedName = uniqueName('Updated');
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(updatedName);

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Updated name should appear
    await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteBudgetItemViaApi(page, orgId, item.id);
  });

  test('should delete a budget item via UI', async ({ page }) => {
    // Setup: create budget item via API
    const itemName = uniqueName('DelBudget');
    await createBudgetItemViaApi(page, orgId, {
      name: itemName,
      category: 'income',
    });

    // Reload to see the item
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(itemName)).toBeVisible({ timeout: 10000 });

    // Click delete button on the item's row
    const row = page.getByRole('row').filter({ hasText: itemName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Budget item should disappear
    await expect(page.getByText(itemName)).not.toBeVisible({ timeout: 10000 });
  });
});
