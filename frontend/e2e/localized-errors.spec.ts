import { test, expect, type Page } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createSectionViaApi,
  deleteSectionViaApi,
  uniqueName,
} from './utils/test-helpers';

/**
 * The German a user sees is the German the server sent.
 *
 * Every piece of this was already tested on its own — the catalogue is complete,
 * the middleware negotiates a language, the client sends Accept-Language from
 * the app's own locale rather than the browser's — and no test joined them up.
 * That gap is exactly where the original defect lived: a German user was shown
 * strictly less about a failure than an English one, because the UI had a
 * catalogue of its own and fell back to a generic sentence whenever it had no
 * entry.
 *
 * There is no frontend catalogue any more, so the only way a German user can see
 * anything specific is if the server localized it. This spec asserts that end to
 * end, against the response body rather than against a string copied into the
 * test — a hardcoded expectation would pass just as happily if the frontend had
 * quietly re-grown its own translations.
 *
 * A conflict is the trigger because it is a message no client-side rule can
 * produce: the name is only known to be taken once the server says so.
 *
 * Desktop only. What is under test is the language of the text, and that does
 * not vary by viewport — the layout of the surface it appears on is covered by
 * form-error-summary.spec.ts and dialog-layout.spec.ts.
 */
test.describe('Localized error messages', () => {
  // One project is enough: what is under test is the language of the text, and
  // that does not vary by viewport. Running it three times would add a minute to
  // every CI run for no extra signal.
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'chromium', 'language does not vary by viewport');
  });

  let orgId: number;
  let sectionId: number;
  let takenName: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await createTestOrg(page, 'Localized')).orgId;
    takenName = uniqueName('Bereich');
    sectionId = (await createSectionViaApi(page, orgId, takenName)).id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteSectionViaApi(page, orgId, sectionId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  /**
   * Switches the app's language the way a user does. The locale lives in a
   * cookie the header writes, not in the browser's Accept-Language, so setting
   * the context locale would not exercise the path the client actually uses.
   */
  async function switchLanguage(page: Page, name: 'English' | 'Deutsch') {
    await page.getByRole('button', { name: /sprache|language/i }).click();
    await page.getByRole('menuitem', { name }).click();
    await expect
      .poll(() => page.evaluate(() => document.cookie))
      .toContain(`locale=${name === 'Deutsch' ? 'de' : 'en'}`);
  }

  /**
   * Submits the taken section name and returns the rejected response alongside
   * the problem document it carried.
   */
  async function submitDuplicateName(page: Page) {
    await page.getByRole('tab', { name: /verwalten|manage/i }).click({ force: true });
    await page.getByRole('button', { name: /neuer bereich|new section/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByLabel(/name/i).fill(takenName);

    const rejected = page.waitForResponse(
      (r) => r.request().method() === 'POST' && r.url().includes('/sections') && r.status() === 409
    );
    await dialog.getByRole('button', { name: /speichern|save/i }).click();

    const response = await rejected;
    return { response, problem: await response.json() };
  }

  test('a German user reads the German the server sent, not the English', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('load');
    await switchLanguage(page, 'Deutsch');
    // The switch takes effect on the next render, so wait for the page itself to
    // be German before trusting anything that follows.
    await expect(page.getByRole('tab', { name: 'Verwalten' })).toBeVisible();

    const { response, problem } = await submitDuplicateName(page);

    // The body carries both languages, and says so.
    expect(response.headers()['content-language']).toBe('en, de');
    expect(
      problem.localized,
      'a German request must come back with a localized member'
    ).toBeTruthy();
    expect(problem.localized.locale).toBe('de');
    expect(problem.detail, 'the English member stays, for logs and support').toBeTruthy();
    expect(
      problem.localized.detail,
      'a localized detail identical to the English one means nothing was translated'
    ).not.toBe(problem.detail);

    // And it is the German one that reaches the user. This is the assertion the
    // original defect would have failed: the German text is as specific as the
    // English, naming the same condition rather than a generic "an error
    // occurred".
    await expect(page.getByText(problem.localized.detail, { exact: false }).first()).toBeVisible();
    await expect(page.getByText(problem.detail, { exact: false })).toHaveCount(0);
  });

  test('an English user gets the English, with nothing extra attached', async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('load');
    await switchLanguage(page, 'English');
    await expect(page.getByRole('tab', { name: 'Manage' })).toBeVisible();

    const { response, problem } = await submitDuplicateName(page);

    // No localized member for English: duplicating the English text under
    // another key would claim a translation happened when none did.
    expect(response.headers()['content-language']).toBe('en');
    expect(problem.localized).toBeUndefined();
    await expect(page.getByText(problem.detail, { exact: false }).first()).toBeVisible();
  });
});
