import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createChildWithContractViaApi,
  deleteChildViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Children', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Children');
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
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');
  });

  test('should display children list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /children/i }).first()).toBeVisible();
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

    // Select gender — the combobox is inside a container with "Gender" text
    const dialog = page.getByRole('dialog');
    await dialog.locator(':has(> :text("Gender")) >> role=combobox').first().click();
    await page.getByRole('option', { name: /female/i }).click();

    // Fill birthdate
    await page.getByLabel(/birthdate/i).fill('2022-03-15');

    // Fill contract start date
    await page.getByLabel(/start date/i).fill('2024-01-01');

    // Select section via the named combobox
    await dialog.getByRole('combobox', { name: /section/i }).click();
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
    await page.waitForLoadState('load');
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
    await page.waitForLoadState('load');
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

  // A Zurückstellung is recorded on the child rather than inferred from their
  // birthdate. This goes through the edit dialog on purpose: the field crosses
  // the wire as RFC3339, and the bare "YYYY-MM-DD" a date input produces is
  // rejected outright by the Go decoder. That break passes every unit test and
  // every typecheck on both sides -- it only shows up here, where a real form
  // posts a real payload.
  test('records a school entry date and reverses it again', async ({ page }) => {
    const firstName = uniqueName('Deferred');
    await createChildWithContractViaApi(page, orgId, {
      first_name: firstName,
      last_name: 'Sonnenschein',
      gender: 'female',
      // Fixed birthdate so nothing here depends on today: this child's computed
      // school year follows from it, not from when the suite runs.
      birthdate: '2019-06-10',
    });

    await page.reload();
    await page.waitForLoadState('load');
    await page.getByRole('textbox', { name: /search/i }).fill(firstName);
    await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: firstName });
    const dialog = page.getByRole('dialog');
    const schoolEntry = page.getByLabel(/school entry date/i);

    await row.getByRole('button', { name: /edit/i }).click();
    await expect(dialog).toBeVisible({ timeout: 5000 });
    // Empty until somebody records one -- the date is computed by default.
    await expect(schoolEntry).toHaveValue('');

    await schoolEntry.fill('2026-08-01');
    await page.getByRole('button', { name: /save/i }).click();
    // A rejected payload leaves the dialog open with the error on it, so the
    // dialog closing is itself the assertion that the request was accepted.
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    await row.getByRole('button', { name: /edit/i }).click();
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await expect(schoolEntry).toHaveValue('2026-08-01');

    // Clearing it has to reverse the deferral. Only an explicit null does that;
    // an omitted field reads as "leave it alone" and the date would survive.
    await schoolEntry.fill('');
    await page.getByRole('button', { name: /save/i }).click();
    await expect(dialog).not.toBeVisible({ timeout: 10000 });

    await row.getByRole('button', { name: /edit/i }).click();
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await expect(schoolEntry).toHaveValue('');
  });
});
