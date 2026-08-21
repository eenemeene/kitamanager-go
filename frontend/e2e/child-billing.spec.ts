import { test, expect } from '@playwright/test';
import { login, getFirstOrganization, getChildrenViaApi } from './utils/test-helpers';

/**
 * The billing history for one child: what the Senate billed against what
 * KitaManager expected.
 *
 * The route had no test at all. It is also the page a Kita reads when the money
 * does not add up, so "it rendered" is not a useful bar -- the numbers and the
 * per-month verdict are the feature.
 *
 * The child under test is chosen by asking the API which one actually has
 * billing rows, rather than taking the first in the list. Picking blind would
 * mean an empty table quietly satisfying every assertion below.
 */
test.use({ locale: 'en-US' });

test.describe('Child billing history', () => {
  let orgId: number;
  let childId: number;
  let childName: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await getFirstOrganization(page)).id;

    const children = await getChildrenViaApi(page, orgId);
    for (const child of children.slice(0, 12)) {
      const history = await page.evaluate(
        async ({ o, c }) => {
          const r = await fetch(`/api/v1/organizations/${o}/children/${c}/billing-history`, {
            credentials: 'same-origin',
          });
          return r.ok ? await r.json() : null;
        },
        { o: orgId, c: child.id }
      );
      if (history?.entries?.length) {
        childId = child.id;
        childName = history.child_name;
        break;
      }
    }
    await page.close();
    expect(childId, 'seed data should include a child with billing months').toBeTruthy();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/children/${childId}/billing`);
    await page.waitForLoadState('load');
  });

  test('shows what was billed against what was expected, month by month', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /billing history/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // One row per billed month, each carrying both figures and a verdict.
    const rows = page.locator('table tbody tr');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    const count = await rows.count();
    expect(
      count,
      'the chosen child has billing months, so the table must have rows'
    ).toBeGreaterThan(0);

    const first = rows.first();
    // Currency, not merely text: a row that renders a dash for both figures
    // tells the reader nothing, and is what a broken join looks like.
    await expect(first).toContainText(/(?:€\s*[\d.,]+|[\d.,]+\s*€)/);
  });

  test('summarises the difference, and says which way it points', async ({ page }) => {
    // The number a Kita actually acts on. The sign convention is explained on
    // the page because "difference" alone is ambiguous about who owes whom.
    await expect(page.getByText(/billed total/i).first()).toBeVisible({ timeout: 10000 });
    const difference = page.getByText(/^difference$/i).first();
    await expect(difference).toBeVisible();
    await expect(page.getByText(/positive = billed more than expected/i)).toBeVisible();
  });

  test('is reachable from the children list, not only by typing a URL', async ({ page }) => {
    // The entry point is an icon button in the child's row, not a link -- which
    // is worth pinning, because an icon with no text is exactly the control that
    // gets dropped in a redesign without anyone noticing the page it led to.
    // It is hidden below `sm`, so this asserts the desktop and tablet path.
    test.skip(
      test.info().project.name === 'mobile-chrome',
      'the billing action is hidden below sm; the phone reaches this page from the dashboard'
    );

    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');

    // The list paginates, and the child with billing history is not reliably on
    // the first page. Search for them rather than assuming a position.
    await page.getByRole('textbox', { name: /search/i }).fill(childName);

    // Only the visible one: the page renders a table and a card list, and both
    // are in the DOM with CSS deciding which the viewport shows.
    const row = page
      .locator('tr:visible, [data-slot="card"]:visible')
      .filter({ hasText: childName })
      .first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.getByRole('button', { name: /billing history/i }).click();

    await expect(page).toHaveURL(new RegExp(`/children/${childId}/billing$`), { timeout: 10000 });
    await expect(page.getByRole('heading', { name: /billing history/i }).first()).toBeVisible();
  });
});
