import { test, expect, type Page } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  deleteChildViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

/**
 * The Create Child dialog, end to end: the happy path, the ways it is refused,
 * and what is left behind when it is.
 *
 * The unit suite mocks the API client, so it can prove which request was sent
 * but never what the database ended up holding. Everything here that matters --
 * whether a rejected contract leaves a childless record, whether a form
 * complains before the user has had a chance to be right -- is only observable
 * with a real server, which is why it lives at this level.
 */
test.describe('Create Child dialog', () => {
  let orgId: number;
  const created: number[] = [];

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    ({ orgId } = await createTestOrg(page, 'CreateChildDialog'));
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    for (const id of created) {
      await deleteChildViaApi(page, orgId, id).catch(() => {});
    }
    await deleteTestOrg(page, orgId).catch(() => {});
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');
  });

  async function openDialog(page: Page) {
    await page.getByRole('button', { name: /new child/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5000 });
    return dialog;
  }

  /** Fills every field the form requires. Dates are what a date input yields. */
  async function fillValid(
    page: Page,
    dialog: ReturnType<Page['getByRole']>,
    over: { birthdate?: string; contractFrom?: string } = {}
  ) {
    const firstName = uniqueName('Kind');
    await page.getByLabel(/first name/i).fill(firstName);
    await page.getByLabel(/last name/i).fill('Beispiel');
    await page.getByLabel(/birthdate/i).fill(over.birthdate ?? '2022-03-15');
    await page.getByLabel(/start date/i).fill(over.contractFrom ?? '2024-01-01');
    await dialog.getByRole('combobox', { name: /section/i }).click();
    await page.getByRole('option').first().click();
    return firstName;
  }

  // --- happy path ---------------------------------------------------------

  test('creates the child and their first contract in one request', async ({ page }) => {
    const dialog = await openDialog(page);
    const firstName = await fillValid(page, dialog);

    const responsePromise = page.waitForResponse(
      (r) => r.url().includes('/children') && r.request().method() === 'POST'
    );
    await page.getByRole('button', { name: /save/i }).click();

    const response = await responsePromise;
    expect(response.status()).toBe(201);
    const body = await response.json();
    created.push(body.id);

    // One request carried both, so the contract comes back on the child.
    expect(body.contracts).toHaveLength(1);

    await expect(dialog).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });
  });

  // --- error paths --------------------------------------------------------

  test('refuses an empty form and says how many things are wrong', async ({ page }) => {
    const dialog = await openDialog(page);

    await page
      .getByRole('button', { name: /^(save|create)/i })
      .first()
      .click();

    await expect(dialog.getByTestId('form-error-summary')).toBeVisible();
    // Still open: a refused submit must not discard what was typed.
    await expect(dialog).toBeVisible();
  });

  test('refuses a contract that starts before the child was born', async ({ page }) => {
    // A cross-field rule, so it cannot be caught by any single input.
    const dialog = await openDialog(page);
    await fillValid(page, dialog, { birthdate: '2024-06-01', contractFrom: '2023-01-01' });

    await page.getByRole('button', { name: /save/i }).click();

    await expect(dialog.getByTestId('form-error-summary')).toBeVisible();
    await expect(dialog).toBeVisible();
  });

  test('accepts the form once the problem is corrected', async ({ page }) => {
    // The recovery path: an error must clear when the user fixes it, not
    // survive until some later submit.
    const dialog = await openDialog(page);
    const firstName = await fillValid(page, dialog, {
      birthdate: '2024-06-01',
      contractFrom: '2023-01-01',
    });
    await page.getByRole('button', { name: /save/i }).click();
    await expect(dialog.getByTestId('form-error-summary')).toBeVisible();

    await page.getByLabel(/start date/i).fill('2024-09-01');
    const responsePromise = page.waitForResponse(
      (r) => r.url().includes('/children') && r.request().method() === 'POST'
    );
    await page.getByRole('button', { name: /save/i }).click();

    const response = await responsePromise;
    expect(response.status()).toBe(201);
    created.push((await response.json()).id);
    await expect(dialog).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });
  });

  test('does not complain about a field the user has not reached', async ({ page }) => {
    // The dialog used to focus First Name on open, so the first click anywhere
    // else blurred it and it was judged for a value nobody had been asked for.
    // Adding a property first answered with "First name is required" above a
    // form barely started.
    const dialog = await openDialog(page);

    const suggestion = dialog.getByTestId('property-suggestions').getByRole('button').first();
    if (await suggestion.isVisible().catch(() => false)) {
      await suggestion.click();
      await expect(dialog.getByTestId('form-error-summary')).toBeHidden();
    }

    // Whatever the org's funding offers, opening the dialog alone must be quiet.
    await expect(dialog.getByTestId('form-error-summary')).toBeHidden();
  });

  // --- what a refusal leaves behind ---------------------------------------

  test('leaves no child behind when the contract is rejected', async ({ page }) => {
    // The reason child and contract are one request. Driven through the API
    // because the form blocks this client-side -- the point is what the server
    // does when it is asked anyway, and the answer has to be "nothing".
    const firstName = uniqueName('Orphan');

    const status = await page.evaluate(
      async ({ orgId, firstName }) => {
        const csrf = document.cookie.match(/csrf_token=([^;]+)/)?.[1] ?? '';
        const res = await fetch(`/api/v1/organizations/${orgId}/children`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
          credentials: 'same-origin',
          body: JSON.stringify({
            first_name: firstName,
            last_name: 'Beispiel',
            gender: 'female',
            birthdate: '2022-03-15',
            // No such section in this organization.
            contract: { from: '2024-01-01T00:00:00Z', section_id: 99999999 },
          }),
        });
        return res.status;
      },
      { orgId, firstName }
    );
    expect(status).toBeGreaterThanOrEqual(400);

    // The child must not exist -- and proving that needs a view an orphan cannot
    // hide from. The list endpoint defaults `active_on` to today, so a child
    // with no contract is filtered out of it and this assertion would pass
    // whether or not the row survived. The YAML export is documented as the
    // opposite: no date default, every child. That an orphan is invisible to the
    // ordinary list is exactly what let these accumulate unnoticed.
    const exported = await page.evaluate(
      async ({ orgId }) => {
        const res = await fetch(`/api/v1/organizations/${orgId}/children/export/yaml`, {
          credentials: 'same-origin',
        });
        return res.text();
      },
      { orgId }
    );
    expect(exported).not.toContain(firstName);
  });
});
