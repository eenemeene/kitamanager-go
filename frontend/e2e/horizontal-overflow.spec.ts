import { test, expect } from '@playwright/test';
import { login } from './utils/test-helpers';

/**
 * No page may scroll horizontally.
 *
 * This is a phone-width invariant with a disproportionate blast radius. When any
 * unclipped element is wider than the viewport, mobile Chrome widens the layout
 * viewport to fit it — the children and employees pages expanded 412px to 480px,
 * attendance to 484px. Every page then scrolls sideways, and `100vw` resolves to
 * the widened value, so anything sized against it (dialogs, fixed bars) is
 * mis-measured and mis-centred. One over-wide toolbar mis-lays-out the whole app.
 *
 * The cause was a stepper row: four 44px icon buttons, a long date label and a
 * "Today" button came to 419px on one non-wrapping line. Wide *content* is fine
 * — tables live in `overflow-x: auto` wrappers and scroll within themselves,
 * which is why this asserts the document's scrollWidth rather than hunting for
 * wide elements.
 *
 * Runs in every project: the same assertion is trivially true on desktop and
 * cheap to evaluate, and a tablet regression would otherwise go unnoticed until
 * someone opened one.
 */
test.describe('Horizontal overflow', () => {
  const paths = (org: string) => [
    '/organizations',
    `/organizations/${org}/children`,
    `/organizations/${org}/employees`,
    `/organizations/${org}/sections`,
    `/organizations/${org}/attendance`,
  ];

  test('no page scrolls horizontally', async ({ page }) => {
    await login(page);
    await page.goto('/organizations');
    await page.waitForLoadState('load');
    const href = await page.locator('a[href*="/organizations/"]').first().getAttribute('href');
    const org = (href || '/organizations/1').split('/')[2];

    for (const path of paths(org)) {
      await page.goto(path);
      await page.waitForLoadState('load');
      // The toolbars this guards against render after their data arrives, so a
      // measurement taken too early would pass on an empty page.
      await page.waitForTimeout(700);

      const { client, scroll } = await page.evaluate(() => ({
        client: document.documentElement.clientWidth,
        scroll: document.documentElement.scrollWidth,
      }));

      expect(
        scroll,
        `${path} scrolls horizontally (scrollWidth ${scroll} > viewport ${client}). ` +
          `Something unclipped is wider than the viewport; wide content belongs in an ` +
          `overflow-x:auto container.`
      ).toBeLessThanOrEqual(client);
    }
  });
});
