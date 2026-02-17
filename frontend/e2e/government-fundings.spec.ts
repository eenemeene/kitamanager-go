import { test, expect } from '@playwright/test';
import {
  login,
  createGovernmentFundingViaApi,
  deleteGovernmentFundingViaApi,
  getGovernmentFundingsViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Government Fundings', () => {
  // Tests must run serially: all share the same single berlin funding (unique state constraint)
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/government-fundings');
    await page.waitForLoadState('networkidle');
  });

  test('should display government fundings list', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: /government funding/i }).first()
    ).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new government funding via UI', async ({ page }) => {
    // Government fundings have a unique constraint on state (only "berlin" allowed).
    // Delete any existing berlin funding first, then create via UI.
    const existingFundings = await getGovernmentFundingsViaApi(page);
    const berlinFunding = existingFundings.find((f) => f.name);
    const savedName = berlinFunding?.name;
    if (berlinFunding) {
      await deleteGovernmentFundingViaApi(page, berlinFunding.id);
    }

    const fundingName = uniqueName('TestFunding');

    // Reload page after deleting existing funding
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Click "New Government Funding" button
    await page.getByRole('button', { name: /new government funding/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form fields
    await page.getByLabel(/name/i).fill(fundingName);

    // State defaults to "Berlin" (the only option), no need to select

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Funding should appear in list
    await expect(page.getByText(fundingName)).toBeVisible({ timeout: 10000 });

    // Cleanup: delete the test funding and restore original
    const fundings = await getGovernmentFundingsViaApi(page);
    const created = fundings.find((f) => f.name === fundingName);
    if (created) {
      await deleteGovernmentFundingViaApi(page, created.id);
    }
    // Restore original funding
    if (savedName) {
      await createGovernmentFundingViaApi(page, { name: savedName, state: 'berlin' });
    }
  });

  test('should edit a government funding via UI', async ({ page }) => {
    // Use the existing seeded funding
    const fundings = await getGovernmentFundingsViaApi(page);
    expect(fundings.length).toBeGreaterThan(0);
    const funding = fundings[0];
    const origName = funding.name;

    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    // Click edit button on the funding's row
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

    // Revert the name back to original
    const row2 = page.getByRole('row').filter({ hasText: updatedName });
    await row2.getByRole('button', { name: /edit/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(origName);
    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });
  });

  test('should delete a government funding via UI', async ({ page }) => {
    // Create a fresh funding for deletion (need to remove existing one first due to unique state)
    const existingFundings = await getGovernmentFundingsViaApi(page);
    const berlinFunding = existingFundings.find((f) => f.name);
    const savedName = berlinFunding?.name;
    if (berlinFunding) {
      await deleteGovernmentFundingViaApi(page, berlinFunding.id);
    }

    // Create a disposable funding
    const fundingName = uniqueName('DelFunding');
    await createGovernmentFundingViaApi(page, { name: fundingName, state: 'berlin' });

    // Reload to see it
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(fundingName)).toBeVisible({ timeout: 10000 });

    // Click delete button on the funding's row
    const row = page.getByRole('row').filter({ hasText: fundingName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Funding should disappear
    await expect(page.getByText(fundingName)).not.toBeVisible({ timeout: 10000 });

    // Restore original funding
    if (savedName) {
      await createGovernmentFundingViaApi(page, { name: savedName, state: 'berlin' });
    }
  });
});
