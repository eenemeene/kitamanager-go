import { test, expect, type Page } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createPayPlanViaApi,
  seedPayPlanCoverageViaApi,
  createEmployeeWithContractViaApi,
  deleteEmployeeViaApi,
  getEmployeesViaApi,
  uniqueName,
} from './utils/test-helpers';
import { skipWithoutRowActions } from './utils/viewport';

/**
 * Every action the employees list offers for an employee that already exists.
 *
 * The row renders four buttons — contract history, add contract, edit, delete —
 * and the toolbar three more that act on the employees already in the list:
 * export Excel (which carries the list's own search and filters), export YAML,
 * import YAML. This file drives all seven, plus the staff-category filter that
 * decides which employees the row actions are reachable for at all.
 *
 * `employees.spec.ts` already drives edit-and-save and delete-and-confirm as the
 * list's CRUD arc, so those two happy paths are not repeated here. What this file
 * adds for them is the part a happy path never reaches: that the edit dialog opens
 * carrying the employee's current values rather than a blank form, that the delete
 * confirmation names who is about to be deleted, and that cancelling either one
 * changes nothing.
 */

test.use({ locale: 'en-US' });

let orgId: number;

test.beforeAll(async ({ browser }) => {
  const page = await browser.newPage();
  await login(page);
  const testOrg = await createTestOrg(page, 'EmpActions');
  orgId = testOrg.orgId;
  // Contract create validates the pinned (grade, step) against the pay plan, so
  // the plan needs an entry for every combination the dialog can submit.
  const payplan = await createPayPlanViaApi(page, orgId, 'Test Pay Plan');
  await seedPayPlanCoverageViaApi(page, orgId, payplan.id, { from: '2020-01-01' });
  await page.close();
});

test.afterAll(async ({ browser }) => {
  const page = await browser.newPage();
  await login(page);
  await deleteTestOrg(page, orgId);
  await page.close();
});

/**
 * Seeds one employee with an active contract and leaves the list filtered down
 * to it, so the returned row is the only one the assertions can match.
 *
 * The list filters by `active_on=today`, which is why the contract is not
 * optional: an employee without one is created successfully and then does not
 * appear.
 */
async function seedAndFind(page: Page, prefix: string) {
  const firstName = uniqueName(prefix);
  const employee = await createEmployeeWithContractViaApi(page, orgId, {
    first_name: firstName,
    last_name: 'Actions',
    gender: 'female',
    birthdate: '1988-04-11',
  });

  await page.goto(`/organizations/${orgId}/employees`);
  await page.waitForLoadState('load');

  // The search box is debounced, and the export buttons carry whatever the
  // filters held at the moment they were clicked. Seeing the row is therefore
  // not enough to know the search has landed -- with one employee in the
  // organization the row is there either way -- so wait for the request that
  // actually carries it.
  const filtered = page.waitForResponse(
    (resp) => resp.url().includes('/employees?') && resp.url().includes('search=')
  );
  await page.getByRole('textbox', { name: /search/i }).fill(firstName);
  await filtered;
  await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });

  const row = page.getByRole('row').filter({ hasText: firstName });
  return { employee, firstName, row };
}

test.describe('Employee row actions', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('contract history opens that employee history page', async ({ page }) => {
    skipWithoutRowActions(test, page, 'Contract History');

    const { employee, firstName, row } = await seedAndFind(page, 'HistBtn');
    try {
      await row.getByRole('button', { name: /contract history/i }).click();

      await expect(page).toHaveURL(
        new RegExp(`/organizations/${orgId}/employees/${employee.id}/contracts`)
      );
      await expect(page.getByRole('heading', { name: /contract history/i })).toBeVisible({
        timeout: 10000,
      });
      // The breadcrumb is what proves the click carried the right employee
      // through, not just that some history page opened.
      await expect(page.getByText(`${firstName} Actions`)).toBeVisible();
      await expect(page.locator('tbody tr')).toHaveCount(1);
      await expect(page.getByText('S8a / 1')).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('add contract opens prefilled from the active contract and amends it', async ({ page }) => {
    skipWithoutRowActions(test, page, 'Add Contract');

    const { employee, firstName, row } = await seedAndFind(page, 'AmendBtn');
    try {
      await row.getByRole('button', { name: /add contract/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 5000 });
      await expect(dialog.getByText(`New Contract for ${firstName} Actions`)).toBeVisible();

      // Amend mode: the dialog announces the contract it is about to end and
      // offers to end it, ticked, because that is the common case.
      await expect(page.getByText(/this employee has an active contract/i)).toBeVisible();
      await expect(page.locator('#endCurrentContract')).toBeChecked();

      // Carried over from the active contract so the user only edits what changed.
      // The start date is deliberately not asserted: it defaults to tomorrow.
      await expect(page.locator('#grade')).toHaveValue('S8a');
      await expect(page.locator('#step')).toHaveValue('1');
      await expect(page.locator('#weekly_hours')).toHaveValue('39');
      await expect(page.getByRole('combobox', { name: /staff category/i })).toContainText(
        /qualified staff/i
      );

      await page.locator('#weekly_hours').clear();
      await page.locator('#weekly_hours').fill('20');
      await page.getByRole('button', { name: /save/i }).click();

      await expect(dialog).not.toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/previous contract ended/i).first()).toBeVisible({
        timeout: 10000,
      });

      // Both contracts on record: the old hours for the months they applied to,
      // the new ones from the effective date. That is the whole point of amending
      // rather than editing in place.
      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('load');
      await expect(page.locator('tbody tr')).toHaveCount(2, { timeout: 10000 });
      await expect(page.getByText('39h')).toBeVisible();
      await expect(page.getByText('20h')).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('add contract with the end-current box cleared is refused as an overlap', async ({
    page,
  }) => {
    skipWithoutRowActions(test, page, 'Add Contract');

    const { employee, row } = await seedAndFind(page, 'NoEndBtn');
    try {
      await row.getByRole('button', { name: /add contract/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 5000 });

      // Clearing the box asks for a plain second contract instead of an
      // amendment. The active one runs open-ended, so any second contract
      // overlaps it — the server refuses, and the refusal has to reach the user
      // with the dialog still holding what they typed.
      await page.locator('#endCurrentContract').uncheck();
      await expect(page.locator('#endCurrentContract')).not.toBeChecked();

      const responsePromise = page.waitForResponse(
        (resp) =>
          resp.url().includes(`/employees/${employee.id}/contracts`) &&
          resp.request().method() === 'POST'
      );
      await page.getByRole('button', { name: /save/i }).click();

      expect((await responsePromise).status()).toBe(409);
      await expect(dialog).toBeVisible();
      await expect(page.getByText(/overlap/i).first()).toBeVisible({ timeout: 10000 });

      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('load');
      await expect(page.locator('tbody tr')).toHaveCount(1, { timeout: 10000 });
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('cancelling the contract dialog leaves the contracts untouched', async ({ page }) => {
    skipWithoutRowActions(test, page, 'Add Contract');

    const { employee, row } = await seedAndFind(page, 'CancelContract');
    try {
      await row.getByRole('button', { name: /add contract/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 5000 });
      await page.locator('#weekly_hours').clear();
      await page.locator('#weekly_hours').fill('12');
      await dialog.getByRole('button', { name: /cancel/i }).click();

      await expect(dialog).not.toBeVisible({ timeout: 5000 });

      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('load');
      await expect(page.locator('tbody tr')).toHaveCount(1, { timeout: 10000 });
      await expect(page.getByText('39h')).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('edit opens carrying the employee current values', async ({ page }) => {
    const { employee, firstName, row } = await seedAndFind(page, 'EditPrefill');
    try {
      await row.getByRole('button', { name: /^edit$/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 5000 });
      await expect(dialog.getByText(/edit employee/i)).toBeVisible();

      // A create dialog reused for editing is only correct if it arrives filled:
      // a blank birthdate saved over a real one is a silent data loss.
      await expect(page.getByLabel(/first name/i)).toHaveValue(firstName);
      await expect(page.getByLabel(/last name/i)).toHaveValue('Actions');
      await expect(page.getByLabel(/birthdate/i)).toHaveValue('1988-04-11');
      await expect(page.getByRole('combobox', { name: /gender/i })).toContainText(/female/i);
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('cancelling the edit dialog discards the change', async ({ page }) => {
    const { employee, firstName, row } = await seedAndFind(page, 'EditCancel');
    try {
      await row.getByRole('button', { name: /^edit$/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 5000 });
      await page.getByLabel(/last name/i).clear();
      await page.getByLabel(/last name/i).fill('Discarded');
      await dialog.getByRole('button', { name: /cancel/i }).click();

      await expect(dialog).not.toBeVisible({ timeout: 5000 });
      await expect(page.getByText(`${firstName} Actions`)).toBeVisible({ timeout: 10000 });
      await expect(page.getByText('Discarded')).toHaveCount(0);

      // Asserted against the API too: the list could just be serving a cache.
      const stored = await getEmployeesViaApi(page, orgId, { search: firstName });
      expect(stored.map((e) => e.last_name)).toEqual(['Actions']);
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('delete asks for confirmation naming the employee', async ({ page }) => {
    const { employee, firstName, row } = await seedAndFind(page, 'DelConfirm');
    try {
      await row.getByRole('button', { name: /^delete$/i }).click();

      const confirm = page.getByRole('alertdialog');
      await expect(confirm).toBeVisible({ timeout: 5000 });
      // Naming who is about to go is the only thing standing between a misclick
      // and a deleted employee.
      await expect(confirm.getByText(`${firstName} Actions`)).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('cancelling the delete confirmation keeps the employee', async ({ page }) => {
    const { employee, firstName, row } = await seedAndFind(page, 'DelCancel');
    try {
      await row.getByRole('button', { name: /^delete$/i }).click();

      const confirm = page.getByRole('alertdialog');
      await expect(confirm).toBeVisible({ timeout: 5000 });
      await confirm.getByRole('button', { name: /cancel/i }).click();

      await expect(confirm).not.toBeVisible({ timeout: 5000 });
      await expect(page.getByText(firstName)).toBeVisible();

      const stored = await getEmployeesViaApi(page, orgId, { search: firstName });
      expect(stored).toHaveLength(1);
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('confirming the delete removes the employee', async ({ page }) => {
    // No `finally` cleanup here, unlike every other test in this file: the point
    // of this one is that the employee is gone, and a second delete would 404 and
    // fail the teardown rather than the test.
    const { firstName, row } = await seedAndFind(page, 'DelConfirmed');
    await row.getByRole('button', { name: /^delete$/i }).click();

    const confirm = page.getByRole('alertdialog');
    await expect(confirm).toBeVisible({ timeout: 5000 });
    await confirm.getByRole('button', { name: /^delete$/i }).click();

    await expect(confirm).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(firstName)).toHaveCount(0);

    const stored = await getEmployeesViaApi(page, orgId, { search: firstName });
    expect(stored).toHaveLength(0);
  });

  test('the staff-category filter narrows the list to matching employees', async ({ page }) => {
    const { employee, firstName } = await seedAndFind(page, 'CatFilter');
    try {
      const filter = page.getByRole('combobox', { name: /filter by category/i });

      // The seeded contract is qualified, so the other two categories have to
      // hide it and its own has to bring it back. Asserting only that the right
      // category shows it would pass against a filter that never filtered.
      await filter.click();
      await page.getByRole('option', { name: /supplementary staff/i }).click();
      await expect(page.getByText(/no results found/i)).toBeVisible({ timeout: 10000 });
      await expect(page.getByText(firstName)).toHaveCount(0);

      await filter.click();
      await page.getByRole('option', { name: /qualified staff/i }).click();
      await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });

      await filter.click();
      await page.getByRole('option', { name: /^all$/i }).click();
      await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });
});

test.describe('Employee list export and import', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('export Excel carries the search the list is showing', async ({ page }) => {
    const { employee, firstName } = await seedAndFind(page, 'ExcelBtn');
    try {
      // The button exports what is on screen, not the whole organization, so the
      // query string is the assertion: a search that does not reach the export is
      // the difference between one row and every employee the Kita has.
      const downloadPromise = page.waitForEvent('download');
      await page.getByRole('button', { name: /export excel/i }).click();
      const download = await downloadPromise;

      expect(download.suggestedFilename()).toBe('mitarbeiter.xlsx');
      const exportUrl = new URL(download.url());
      expect(exportUrl.pathname).toContain(`/organizations/${orgId}/employees/export/excel`);
      expect(exportUrl.searchParams.get('search')).toBe(firstName);
      expect(exportUrl.searchParams.get('active_on')).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('export YAML downloads the employees file', async ({ page }) => {
    const { employee } = await seedAndFind(page, 'YamlBtn');
    try {
      const downloadPromise = page.waitForEvent('download');
      await page.getByRole('button', { name: /export yaml/i }).click();
      const download = await downloadPromise;

      expect(download.suggestedFilename()).toBe('employees.yaml');
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('import YAML upserts the exported employees instead of duplicating them', async ({
    page,
  }) => {
    const { employee, firstName } = await seedAndFind(page, 'ImportBtn');
    try {
      // Round-tripping the organization's own export is the one payload
      // guaranteed to be in whatever shape the importer expects, and it makes
      // the two endpoints assert against each other rather than against a
      // fixture that goes stale when the schema moves.
      const exported = await page.evaluate(async (id) => {
        const resp = await fetch(`/api/v1/organizations/${id}/employees/export/yaml`, {
          credentials: 'same-origin',
        });
        if (!resp.ok) throw new Error(`export failed: ${resp.status}`);
        return resp.text();
      }, orgId);
      expect(exported).toContain(firstName);

      const chooserPromise = page.waitForEvent('filechooser');
      await page.getByRole('button', { name: /import yaml/i }).click();
      const chooser = await chooserPromise;
      await chooser.setFiles({
        name: 'employees.yaml',
        mimeType: 'application/x-yaml',
        buffer: Buffer.from(exported, 'utf8'),
      });

      await expect(page.getByText(/employees created successfully/i).first()).toBeVisible({
        timeout: 15000,
      });

      // Import upserts on name plus birthdate, so re-importing what was just
      // exported has to leave the list the same length it was.
      const stored = await getEmployeesViaApi(page, orgId, { search: firstName });
      expect(stored).toHaveLength(1);
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });
});
