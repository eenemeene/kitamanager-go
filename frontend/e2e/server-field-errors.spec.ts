import { test, expect, type Page } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  uniqueName,
  createPayPlanViaApi,
} from './utils/test-helpers';

/**
 * A field the *server* rejected is marked on the form that submitted it.
 *
 * The client-side rules are covered elsewhere. What is new here is the other
 * direction: the API rejects a field the form did not catch, and the user has to
 * end up looking at that input rather than at a toast that scrolls away.
 *
 * The rejection is mocked, because the forms validate the same fields the
 * backend does -- so a violation the client lets through is, by construction,
 * hard to produce on demand. That makes this a test of the plumbing rather than
 * of any particular rule, which is what changed in this branch.
 *
 * A mocked route that stops matching becomes a test that asserts nothing, which
 * is how the timeline rollback test rotted. Each test here proves the mock
 * actually fired before believing anything it sees.
 */
test.use({ locale: 'en-US' });

interface Rejection {
  field: string;
  reason: string;
  rule?: string;
}

/**
 * Answers the next create with a 400 naming the given fields, and reports
 * whether it was ever asked.
 */
async function rejectCreateWith(page: Page, url: RegExp, params: Rejection[]) {
  const state = { fired: false };
  await page.route(url, async (route) => {
    if (route.request().method() !== 'POST') return route.fallback();
    state.fired = true;
    await route.fulfill({
      status: 400,
      contentType: 'application/problem+json',
      body: JSON.stringify({
        type: 'https://example.test/errors/#validation',
        title: 'Validation failed',
        status: 400,
        code: 'validation',
        detail: 'the request was rejected',
        invalid_params: params.map((p) => ({ rule: 'required', ...p })),
      }),
    });
  });
  return state;
}

test.describe('Server field violations', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await createTestOrg(page, 'FieldErrors')).orgId;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test('the organizations form marks the field the server named', async ({ page }) => {
    const mock = await rejectCreateWith(page, /\/api\/v1\/organizations(\?.*)?$/, [
      { field: 'name', reason: 'is already taken' },
    ]);

    await login(page);
    await page.goto('/organizations');
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new organization/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByLabel('Name', { exact: true }).fill(uniqueName('Rejected'));
    await dialog.locator('[role="combobox"]').first().click();
    await page.getByRole('option', { name: /berlin/i }).click();
    await dialog.getByLabel(/default section name/i).fill('Default');
    await dialog.getByRole('button', { name: /^save/i }).first().click();

    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible({ timeout: 10000 });
    expect(mock.fired, 'the mocked rejection never fired — this test proved nothing').toBe(true);

    // The input itself is marked, so the field is findable without the summary.
    await expect(dialog.locator('#name')).toHaveAttribute('aria-invalid', 'true');
    // And the summary names it the way the form labels it.
    await expect(summary).toContainText('Name');
    // The toast would only repeat what is already on screen.
    await expect(page.locator('[data-state="open"] [role="status"]')).toHaveCount(0);
  });

  test('a violation naming a field the form does not have is still shown', async ({ page }) => {
    // Nothing to mark, so the summary is the only thing standing between the
    // user and a submit that failed for an unstated reason.
    const mock = await rejectCreateWith(page, /\/api\/v1\/organizations(\?.*)?$/, [
      { field: 'billing_reference', reason: 'is not recognised' },
    ]);

    await login(page);
    await page.goto('/organizations');
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new organization/i }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('Name', { exact: true }).fill(uniqueName('Unmapped'));
    await dialog.locator('[role="combobox"]').first().click();
    await page.getByRole('option', { name: /berlin/i }).click();
    await dialog.getByLabel(/default section name/i).fill('Default');
    await dialog.getByRole('button', { name: /^save/i }).first().click();

    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible({ timeout: 10000 });
    expect(mock.fired, 'the mocked rejection never fired — this test proved nothing').toBe(true);
    await expect(summary).toContainText(/billing_reference|not recognised/i);
  });

  test('a pay-plan period form marks the field the server named', async ({ page }) => {
    // A detail page rather than a list dialog, and a mutation that goes through
    // useResourceMutation rather than useCrudMutations -- the two shapes reach
    // the form by the same route now, and this is the half that was not covered.
    const mock = await rejectCreateWith(
      page,
      /\/api\/v1\/organizations\/\d+\/pay-plans\/\d+\/periods$/,
      [{ field: 'weekly_hours', reason: 'must be greater than zero' }]
    );

    await login(page);
    const plan = await createPayPlanViaApi(page, orgId, uniqueName('FieldErrPlan'));
    await page.goto(`/organizations/${orgId}/payplans/${plan.id}`);
    await page.waitForLoadState('load');

    // The detail page opens in table view; adding a period is offered from the
    // panels view, which is the path a user takes to get this dialog at all.
    await page.getByRole('button', { name: /^panels$/i }).click();
    await page.getByRole('button', { name: /add period/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByLabel(/from date/i).fill('2026-01-01');
    await dialog
      .getByRole('button', { name: /^(save|create|add)/i })
      .first()
      .click();

    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible({ timeout: 10000 });
    expect(mock.fired, 'the mocked rejection never fired — this test proved nothing').toBe(true);
    await expect(dialog.locator('#weekly_hours')).toHaveAttribute('aria-invalid', 'true');
    await expect(summary).toContainText(/weekly hours/i);
  });

  test('a rejection with no fields still reaches the user', async ({ page }) => {
    // The rule the suppression depends on: silence here would make a failed
    // save look like a successful one.
    const state = { fired: false };
    await page.route(/\/api\/v1\/organizations(\?.*)?$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback();
      state.fired = true;
      await route.fulfill({
        status: 409,
        contentType: 'application/problem+json',
        body: JSON.stringify({
          type: 'u',
          title: 'Conflict',
          status: 409,
          code: 'conflict',
          detail: 'organization with this name already exists',
        }),
      });
    });

    await login(page);
    await page.goto('/organizations');
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new organization/i }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('Name', { exact: true }).fill(uniqueName('Conflict'));
    await dialog.locator('[role="combobox"]').first().click();
    await page.getByRole('option', { name: /berlin/i }).click();
    await dialog.getByLabel(/default section name/i).fill('Default');
    await dialog.getByRole('button', { name: /^save/i }).first().click();

    await expect(page.getByText(/already exists/i).first()).toBeVisible({ timeout: 10000 });
    expect(state.fired, 'the mocked rejection never fired — this test proved nothing').toBe(true);
  });
});
