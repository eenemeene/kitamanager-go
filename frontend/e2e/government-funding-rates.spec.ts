import { test, expect } from '@playwright/test';
import {
  login,
  createGovernmentFundingViaApi,
  deleteGovernmentFundingViaApi,
  getGovernmentFundingsViaApi,
  getGovernmentFundingViaApi,
  deleteFundingPeriodViaApi,
  ensureFundingHasProperties,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Government Fundings', () => {
  // All government funding tests must run serially: only one Berlin funding can exist (unique state constraint)
  test.describe.configure({ mode: 'serial' });

  // Safety net: always restore Berlin funding with periods/properties after all tests
  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await ensureFundingHasProperties(page);
    await page.close();
  });

  test.describe('CRUD', () => {
    test.beforeEach(async ({ page }) => {
      await login(page);
      await page.goto('/government-funding-rates');
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
      const berlinFunding = existingFundings.find((f) => f.state === 'berlin');
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

      // Cleanup: delete the test funding and restore original with properties
      const fundings = await getGovernmentFundingsViaApi(page);
      const created = fundings.find((f) => f.name === fundingName);
      if (created) {
        await deleteGovernmentFundingViaApi(page, created.id);
      }
      // Restore original funding
      if (savedName) {
        await createGovernmentFundingViaApi(page, { name: savedName, state: 'berlin' });
      }
      await ensureFundingHasProperties(page);
    });

    test('should edit a government funding via UI', async ({ page }) => {
      // Use the existing seeded funding
      const fundings = await getGovernmentFundingsViaApi(page);
      const funding = fundings.find((f) => f.state === 'berlin');
      expect(funding).toBeTruthy();
      const origName = funding!.name;

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
      const berlinFunding = existingFundings.find((f) => f.state === 'berlin');
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

      // Restore original funding with properties
      if (savedName) {
        await createGovernmentFundingViaApi(page, { name: savedName, state: 'berlin' });
      }
      await ensureFundingHasProperties(page);
    });
  });

  test.describe('Detail', () => {
    let fundingId: number;
    let fundingName: string;

    test.beforeAll(async ({ browser }) => {
      const page = await browser.newPage();
      await login(page);
      // Ensure Berlin funding exists (CRUD tests may have deleted/recreated it)
      await ensureFundingHasProperties(page);
      const fundings = await getGovernmentFundingsViaApi(page);
      const berlin = fundings.find((f) => f.state === 'berlin');
      expect(berlin).toBeTruthy();
      fundingId = berlin!.id;
      fundingName = berlin!.name;
      // Delete all existing periods for a clean slate
      const details = await getGovernmentFundingViaApi(page, fundingId);
      if (details.periods) {
        for (const period of details.periods) {
          await deleteFundingPeriodViaApi(page, fundingId, period.id).catch(() => {});
        }
      }
      await page.close();
    });

    test('should display funding detail page', async ({ page }) => {
      await login(page);
      await page.goto(`/government-funding-rates/${fundingId}`);
      await page.waitForLoadState('networkidle');

      // Verify heading shows funding name
      await expect(page.getByRole('heading', { name: fundingName })).toBeVisible({
        timeout: 10000,
      });
    });

    test('should create a period via UI', async ({ page }) => {
      await login(page);
      await page.goto(`/government-funding-rates/${fundingId}`);
      await page.waitForLoadState('networkidle');

      // Click "Add Period" button
      await page.getByRole('button', { name: /add.*period/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Fill form fields - ongoing period starting 2020-01-01 (active today)
      await page.locator('#from').fill('2020-01-01');
      await page.locator('#full_time_weekly_hours').fill('39.1');
      await page.locator('#comment').fill('E2E test period');

      // Submit
      await page.getByRole('button', { name: /save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Verify period appears on the page
      await expect(page.getByText('E2E test period')).toBeVisible({ timeout: 10000 });
    });

    test('should create a property within a period via UI', async ({ page }) => {
      await login(page);
      await page.goto(`/government-funding-rates/${fundingId}`);
      await page.waitForLoadState('networkidle');

      // Wait for the test period to be visible
      await expect(page.getByText('E2E test period')).toBeVisible({ timeout: 10000 });

      // Click "Add Property" button on the test period's card
      const periodCard = page.locator('[class*="card"]').filter({
        hasText: 'E2E test period',
      });
      await periodCard.getByRole('button', { name: /add.*property/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Fill property form fields
      await page.locator('#key').fill('care_type');
      await page.locator('#value').fill('e2e_test');
      await page.locator('#label').fill('E2E Test');
      await page.locator('#payment_euros').fill('1234.56');
      await page.locator('#requirement').fill('0.25');

      // Submit
      await page.getByRole('button', { name: /save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Verify property appears (label is shown instead of raw value)
      // Use exact: true to avoid matching "E2E test period" (case-insensitive substring)
      await expect(page.getByText('E2E Test', { exact: true })).toBeVisible({ timeout: 10000 });
    });

    test('should delete a property via UI', async ({ page }) => {
      await login(page);
      await page.goto(`/government-funding-rates/${fundingId}`);
      await page.waitForLoadState('networkidle');

      // Wait for the test property to be visible (label is shown instead of raw value)
      await expect(page.getByText('E2E Test', { exact: true })).toBeVisible({ timeout: 10000 });

      // Navigate to parent div to find the delete button
      await page
        .getByText('E2E Test', { exact: true })
        .locator('..')
        .locator('table button')
        .click();

      // Confirm deletion
      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
      await page.getByRole('button', { name: /delete/i }).click();

      // Property should disappear
      await expect(page.getByText('E2E Test', { exact: true })).not.toBeVisible({ timeout: 10000 });
    });

    test('should delete a period via UI', async ({ page }) => {
      await login(page);
      await page.goto(`/government-funding-rates/${fundingId}`);
      await page.waitForLoadState('networkidle');

      // Wait for the test period to be visible
      await expect(page.getByText('E2E test period')).toBeVisible({ timeout: 10000 });

      // Click the delete (Trash) icon button on the period card
      const periodCard = page.locator('[class*="card"]').filter({
        hasText: 'E2E test period',
      });
      const trashButton = periodCard
        .locator('.flex.gap-2')
        .locator('button')
        .filter({ has: page.locator('svg') })
        .last();
      await trashButton.click();

      // Confirm deletion
      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
      await page.getByRole('button', { name: /delete/i }).click();

      // Period should disappear
      await expect(page.getByText('E2E test period')).not.toBeVisible({ timeout: 10000 });
    });
  });
});
