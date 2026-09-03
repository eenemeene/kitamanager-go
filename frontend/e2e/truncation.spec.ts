import { test, expect, type Page } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createOrganizationViaApi,
  createChildViaApi,
  createChildContractViaApi,
  createEmployeeViaApi,
  getSectionsViaApi,
} from './utils/test-helpers';

/**
 * Long names must stay inside the boxes that hold them.
 *
 * `truncate` looks like it guarantees this and does not. It is
 * `overflow:hidden; text-overflow:ellipsis; white-space:nowrap`, and whether it
 * has any width to clip against depends on the flex chain above it. A flex item
 * whose overflow is hidden gets an automatic minimum size of zero and shrinks
 * happily; one that merely *contains* such an element does not, and pushes its
 * content out of the parent instead.
 *
 * That is how the organisation selector shipped broken: the truncating span sat
 * inside a plain `flex items-center` wrapper, so the wrapper grew to the width
 * of the name and the text ran past the button. It was invisible for as long as
 * the only organisation anyone looked at was called "Kita Sonnenschein".
 *
 * The check is deliberately about the container rather than the text: an
 * element whose own content is wider than itself reports scrollWidth greater
 * than clientWidth. Asserting on the truncating element instead would prove
 * nothing, because a correctly clipped one is *expected* to be wider than its
 * box.
 *
 * This cannot be a jest test. jsdom has no layout engine and reports every
 * width as zero, so the broken and fixed versions look identical to it.
 */

const LONG = 'Kindertagesstätte Regenbogenland am Wasserturm Prenzlauer Berg Nordost';

/** Containers holding truncated text whose content does not fit inside them. */
async function overflowingContainers(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const bad: string[] = [];
    const seen = new Set<Element>();
    for (const el of document.querySelectorAll('.truncate')) {
      // The container is whatever encloses the truncating element. If the text
      // escaped, it is this box that overflows, not the text's own.
      let node: HTMLElement | null = el.parentElement;
      while (node && node !== document.body) {
        if (seen.has(node)) break;
        seen.add(node);
        const overflowed = node.scrollWidth > node.clientWidth + 1;
        const scrolls = ['auto', 'scroll'].includes(getComputedStyle(node).overflowX);
        // A deliberate scroller is allowed to be wider than its viewport; that
        // is what it is for.
        if (overflowed && !scrolls) {
          bad.push(
            `<${node.tagName.toLowerCase()} class="${String(node.className).slice(0, 70)}"> ` +
              `scrollWidth=${node.scrollWidth} clientWidth=${node.clientWidth}`
          );
        }
        node = node.parentElement;
      }
    }
    return [...new Set(bad)];
  });
}

test.describe('Long names stay inside their containers', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    // The organisation name is what broke first, so it is the long one. The
    // suffix keeps it clear of the unique constraint on organizations.name.
    const org = await createOrganizationViaApi(page, `${LONG} ${Date.now()}`, 'berlin', 'Default');
    orgId = org.id;
    const sections = await getSectionsViaApi(page, orgId);

    const child = await createChildViaApi(page, orgId, {
      first_name: 'Maximiliane-Charlotte',
      last_name: 'von Hohenzollern-Sigmaringen-Wittelsbach',
      gender: 'female',
      birthdate: '2022-04-01',
    });
    await createChildContractViaApi(page, orgId, child.id, {
      from: '2024-01-01T00:00:00Z',
      section_id: sections[0].id,
      properties: { care_type: 'ganztag' },
    });
    await createEmployeeViaApi(page, orgId, {
      first_name: 'Friedrich-Wilhelm',
      last_name: 'Schmidt-Hohenlohe-Langenburg',
      gender: 'male',
      birthdate: '1985-03-12',
    });
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteTestOrg(page, orgId).catch(() => {});
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  // The sidebar is on every page, so the organisation name is checked by all of
  // them; the per-page entries are for the cards and lists that carry names of
  // their own.
  for (const path of ['sections', 'children', 'employees', 'users']) {
    test(`${path} keeps long names inside their containers`, async ({ page }) => {
      await page.goto(`/organizations/${orgId}/${path}`);
      await page.waitForLoadState('load');
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

      const overflowing = await overflowingContainers(page);
      expect(
        overflowing,
        `containers overflowed by their own content:\n${overflowing.join('\n')}`
      ).toEqual([]);
    });
  }
});
