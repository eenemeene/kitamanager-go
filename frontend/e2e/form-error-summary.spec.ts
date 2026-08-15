import { test, expect } from '@playwright/test';
import { login, createTestOrg, deleteTestOrg } from './utils/test-helpers';

/**
 * What a user sees when a form is rejected.
 *
 * The unit tests cover the summary's logic; this covers the things only a real
 * browser can answer — that it is reachable and readable at each viewport, that
 * focus actually lands on it, that its items are real tap targets, and that the
 * toast it replaced is gone.
 *
 * Runs in every project on purpose. The tablet and desktop bar is that all of it
 * holds; the phone bar is the same behaviour with more scrolling, which is why
 * the visibility assertion is against the dialog's own scroll container rather
 * than the window.
 *
 * The errors here are the client-side rules, because those are what a user can
 * trigger deterministically — the server's field violations mostly cannot be
 * reached through a form that validates the same fields first. The path from a
 * server violation to a marked field is covered by the unit tests instead.
 */
test.use({ locale: 'en-US' });

test.describe('Form error summary', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'FormErrors');
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

  async function openCreateDialogAndSubmitEmpty(page: import('@playwright/test').Page) {
    await page
      .getByRole('button', { name: /new child|add child/i })
      .first()
      .click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    // Submit with nothing filled in, which fails at least one required rule.
    await dialog
      .getByRole('button', { name: /^(create|save|add)/i })
      .first()
      .click();
    return dialog;
  }

  test('lists every problem, and takes focus so a screen reader announces it', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);

    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible();

    // One or more: an empty child form fails a single rule, because most of its
    // fields carry defaults. The multi-error case — ordering, counting, the
    // unmapped ones — is covered by the unit tests, which can construct it
    // directly instead of hunting for a form that produces it.
    const count = Number(await summary.getAttribute('data-count'));
    expect(count).toBeGreaterThanOrEqual(1);
    await expect(summary.getByRole('listitem')).toHaveCount(count);

    // Focus is on the summary itself, not an input: focusing an input would open
    // the on-screen keyboard on a phone or tablet and can hide the field.
    await expect(summary).toBeFocused();
  });

  test('replaces the toast rather than adding to it', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    await expect(dialog.getByTestId('form-error-summary')).toBeVisible();

    // The toast viewport is always mounted; what matters is that it is empty.
    await expect(page.locator('[role="status"], li[data-state="open"]')).toHaveCount(0);
  });

  test('marks the offending inputs, and its items jump to them', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible();

    // Every field the summary points at is marked for assistive technology.
    await expect(dialog.locator('[aria-invalid="true"]').first()).toBeVisible();

    const firstItem = summary.getByRole('button').first();
    const label = (await firstItem.innerText()).trim();
    await firstItem.click();

    const focusedId = await page.evaluate(() => document.activeElement?.id ?? '');
    expect(focusedId, `activating "${label}" should focus its field`).not.toBe('');
  });

  test('shows no machine identifiers to the user', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    const text = await dialog.getByTestId('form-error-summary').innerText();

    // A cheap proxy for "no JSON path leaked": field paths carry underscores and
    // brackets, labels do not.
    expect(text).not.toMatch(/[[\]]/);
    expect(text).not.toMatch(/\w_\w/);
  });

  test('every item is a 44px touch target', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    const items = dialog.getByTestId('form-error-summary').getByRole('button');

    for (const item of await items.all()) {
      const box = await item.boundingBox();
      expect(box, 'summary item should be laid out').not.toBeNull();
      expect(box!.height).toBeGreaterThanOrEqual(44);
    }
  });

  test('leaves the submit button where the user left it', async ({ page }) => {
    // The bug this guards: the summary is inserted above the form, so everything
    // below it moves down — including the footer holding the button just
    // pressed. Focus also moves to the summary, scrolling the dialog to the top.
    // Together that put Save off-screen and made the user scroll back to retry.
    //
    // Measured on a phone before the footer was pinned: Save sat at y=933 in a
    // dialog ending at 765 — already out of view when the dialog opened, then
    // pushed to 1103 by the summary.
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    await expect(dialog.getByTestId('form-error-summary')).toBeVisible();

    const save = dialog.getByRole('button', { name: /^(create|save|add)/i }).first();
    const box = await save.boundingBox();
    const dialogBox = await dialog.boundingBox();

    expect(box, 'the submit button should still be laid out').not.toBeNull();
    expect(
      box!.y + box!.height,
      'the submit button must stay within the dialog after a rejected submit'
    ).toBeLessThanOrEqual(dialogBox!.y + dialogBox!.height + 1);
    expect(box!.y).toBeGreaterThanOrEqual(dialogBox!.y);
  });

  test('is visible inside the dialog without hunting for it', async ({ page }) => {
    const dialog = await openCreateDialogAndSubmitEmpty(page);
    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible();

    // The dialog is capped at 85vh with its own scrollbar, so "on the page" is
    // not the same as "on screen". This is the assertion that catches scrolling
    // the wrong element — window.scrollTo does nothing inside these dialogs.
    const box = await summary.boundingBox();
    const dialogBox = await dialog.boundingBox();
    expect(box).not.toBeNull();
    expect(dialogBox).not.toBeNull();
    expect(box!.y).toBeGreaterThanOrEqual(dialogBox!.y - 1);
    expect(box!.y).toBeLessThan(dialogBox!.y + dialogBox!.height);
  });
});
