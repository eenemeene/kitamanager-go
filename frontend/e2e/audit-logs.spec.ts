import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createChildViaApi,
  deleteChildViaApi,
  uniqueName,
  ADMIN_EMAIL,
} from './utils/test-helpers';

/**
 * The audit log, from the side of the person who has to read it.
 *
 * The page had no test at all: role-sidebar.spec.ts asserted the nav link exists
 * per role, and nothing ever opened it. Meanwhile the records behind it changed
 * repeatedly -- per-field contract diffs, snapshots of deleted contracts, amend
 * logged as an update plus a create -- all covered by Go tests at the service
 * layer, none of it checked to actually reach a screen.
 *
 * So this asserts the join: an action taken through the API produces a row a
 * human can read, naming who did what to which record, and the filters narrow
 * it rather than merely redrawing.
 */
test.use({ locale: 'en-US' });

test.describe('Audit log', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await createTestOrg(page, 'AuditLog')).orgId;
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
  });

  /**
   * One log entry, whichever way this viewport draws it.
   *
   * Above `md` the page renders a table; on a phone it renders a list of cards
   * instead, with the same three facts on each. Addressing the entry rather than
   * the table is what lets these assertions hold on all three projects -- the
   * first version of this spec looked for a table row and failed on the phone,
   * where there is no table at all.
   */
  function entryFor(page: import('@playwright/test').Page, action: string) {
    // `:visible`, because both presentations are always in the DOM and CSS hides
    // one of them -- without it the phone run matches the table row it cannot see.
    return page.locator('[data-testid="audit-entry"]:visible').filter({ hasText: action });
  }

  test('an action taken elsewhere shows up as a readable row', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('Audited'),
      last_name: 'Child',
      birthdate: '2021-04-02',
      gender: 'female',
    });

    await page.goto(`/organizations/${orgId}/audit-logs`);
    await page.waitForLoadState('load');

    const row = entryFor(page, 'child_create').first();
    await expect(row).toBeVisible({ timeout: 10000 });

    // Who, and which record. A log that says only "something was created" costs
    // the reader the one thing they opened it for.
    await expect(row).toContainText(ADMIN_EMAIL);
    await expect(row).toContainText(`child #${child.id}`);

    await deleteChildViaApi(page, orgId, child.id);
  });

  test('a deletion is recorded as its own event', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('Deleted'),
      last_name: 'Child',
      birthdate: '2020-09-09',
      gender: 'male',
    });
    await deleteChildViaApi(page, orgId, child.id);

    await page.goto(`/organizations/${orgId}/audit-logs`);
    await page.waitForLoadState('load');

    // Both halves of the child's life are on the log, against the same id --
    // the delete is what a reader is most likely to be looking for, and the id
    // is the only way to tie it to the create once the name is gone.
    await expect(entryFor(page, 'child_delete').first()).toBeVisible({ timeout: 10000 });
    await expect(entryFor(page, 'child_delete').first()).toContainText(`child #${child.id}`);
    await expect(entryFor(page, 'child_create').first()).toContainText(`child #${child.id}`);
  });

  test('the action filter narrows the list instead of just redrawing it', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('Filtered'),
      last_name: 'Child',
      birthdate: '2022-02-02',
      gender: 'diverse',
    });

    await page.goto(`/organizations/${orgId}/audit-logs`);
    await page.waitForLoadState('load');
    await expect(entryFor(page, 'child_create').first()).toBeVisible({ timeout: 10000 });

    const filter = page.getByLabel(/action/i);
    await filter.fill('child_create');
    await expect(entryFor(page, 'child_create').first()).toBeVisible({ timeout: 10000 });

    // A term no action contains must empty the table. Without this half, a
    // filter that ignores its input entirely still passes the half above.
    await filter.fill('zzz_no_such_action');
    await expect(page.getByText(/no audit events/i)).toBeVisible({ timeout: 10000 });

    await deleteChildViaApi(page, orgId, child.id);
  });
});
