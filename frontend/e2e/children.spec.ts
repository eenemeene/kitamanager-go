import { test, expect } from '@playwright/test';
import {
  login,
  getFirstOrganization,
  createChildWithContractViaApi,
  deleteChildViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Children', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');
  });

  test('should display children list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /children/i }).first()).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new child via UI', async ({ page }) => {
    const firstName = uniqueName('ChildFirst');
    const lastName = uniqueName('ChildLast');

    // Click "New Child" button
    await page.getByRole('button', { name: /new child/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill personal info fields
    await page.getByLabel(/first name/i).fill(firstName);
    await page.getByLabel(/last name/i).fill(lastName);

    // Select gender via Shadcn Select (first combobox in dialog is gender)
    const comboboxes = page.getByRole('dialog').getByRole('combobox');
    await comboboxes.first().click();
    await page.getByRole('option', { name: /female/i }).click();

    // Fill birthdate
    await page.getByLabel(/birthdate/i).fill('2022-03-15');

    // Fill contract start date
    await page.getByLabel(/start date/i).fill('2024-01-01');

    // Select section (second combobox in dialog)
    await comboboxes.nth(1).click();
    await page.getByRole('option').first().click();

    // Capture the API response
    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/children') && resp.request().method() === 'POST'
    );

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Verify API returned 201 Created
    const response = await responsePromise;
    expect(response.status()).toBe(201);
    const body = await response.json();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Cleanup via API
    await deleteChildViaApi(page, orgId, body.id);
  });

  test('should edit a child via UI', async ({ page }) => {
    // Setup: create child with active contract so it appears in list
    const origFirst = uniqueName('EditChild');
    const child = await createChildWithContractViaApi(page, orgId, {
      first_name: origFirst,
      last_name: 'Original',
      gender: 'male',
      birthdate: '2021-06-10',
    });

    // Reload and search for the child
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByRole('textbox', { name: /search/i }).fill(origFirst);
    await expect(page.getByText(origFirst)).toBeVisible({ timeout: 10000 });

    // Click edit button on the child's row
    const row = page.getByRole('row').filter({ hasText: origFirst });
    await row.getByRole('button', { name: /edit/i }).click();

    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Modify first name
    const updatedFirst = uniqueName('Updated');
    await page.getByLabel(/first name/i).clear();
    await page.getByLabel(/first name/i).fill(updatedFirst);

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Search for updated name
    await page.getByRole('textbox', { name: /search/i }).clear();
    await page.getByRole('textbox', { name: /search/i }).fill(updatedFirst);

    // Updated name should appear
    await expect(page.getByText(updatedFirst)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteChildViaApi(page, orgId, child.id);
  });

  test('should delete a child via UI', async ({ page }) => {
    // Setup: create child with active contract so it appears in list
    const firstName = uniqueName('DelChild');
    await createChildWithContractViaApi(page, orgId, {
      first_name: firstName,
      last_name: 'ToDelete',
      gender: 'female',
      birthdate: '2020-11-20',
    });

    // Reload and search for the child
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByRole('textbox', { name: /search/i }).fill(firstName);
    await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });

    // Click delete button on the child's row
    const row = page.getByRole('row').filter({ hasText: firstName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Child should disappear
    await expect(page.getByText(firstName)).not.toBeVisible({ timeout: 10000 });
  });
});
