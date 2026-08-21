import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { login, getFirstOrganization } from './utils/test-helpers';

/**
 * An axe pass over the real pages, in a real browser.
 *
 * This is the layer the jsdom checks cannot be. `src/components/__tests__/`
 * renders one dialog at a time with mocked data, so it never sees a page with
 * its sidebar, header, filter bar and populated table on screen at once — and
 * more importantly, jsdom computes no styles, so `color-contrast` is skipped
 * there entirely. Here it is a real rendering, which makes this the only thing
 * in the suite that checks the palette against what is actually painted rather
 * than against the numbers in globals.css.
 *
 * No viewport is pinned. The playwright config already runs every spec under
 * three projects — chromium at 1280, tablet-chrome at 768 and mobile-chrome on
 * a Pixel 7 — so these routes get scanned at all three widths without this file
 * knowing about it. Layout-dependent findings (a control that only overlaps at
 * 412px, a contrast pair that only appears once a column is hidden) surface on
 * their own.
 */

test.use({ locale: 'en-US' });

/**
 * Radix parks sentinel spans with `tabindex=0` at the edges of every portal it
 * opens, inside a subtree it marks `aria-hidden`. That is how it keeps focus
 * inside the layer, it is not something this codebase controls, and it trips
 * `aria-hidden-focus` on every dialog, popover and select in the app. Left on,
 * the rule is the only thing this spec would ever report.
 */
const RADIX_FOCUS_GUARDS = 'aria-hidden-focus';

/** The conformance target. Excludes axe's opinionated best-practice rules. */
const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

async function scan(page: Page) {
  return new AxeBuilder({ page }).withTags(WCAG_TAGS).disableRules([RADIX_FOCUS_GUARDS]).analyze();
}

/** Reports every violation at once, with the offending markup. */
function format(results: Awaited<ReturnType<typeof scan>>): string {
  return results.violations
    .map(
      (v) =>
        `[${v.impact}] ${v.id} — ${v.help}\n` +
        `    ${v.helpUrl}\n` +
        v.nodes
          .map(
            (n) =>
              `    ${n.target.join(' ')}\n      ${n.html.slice(0, 200)}\n` +
              // axe's summary carries the measured numbers -- for color-contrast
              // that is the actual ratio and the two colours it compared, which
              // is the difference between a report you can act on and one that
              // sends you back to the browser to measure by hand.
              `      ${n.failureSummary?.replace(/\n/g, '\n      ')}`
          )
          .join('\n')
    )
    .join('\n\n');
}

test.describe('Accessibility', () => {
  test('the login page has no violations', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('load');
    // The form is what this page is for; scanning before it hydrates would pass
    // on an empty card.
    await expect(page.getByRole('button', { name: /sign in|login/i })).toBeVisible({
      timeout: 10000,
    });

    const results = await scan(page);
    expect(results.violations, format(results)).toEqual([]);
  });

  test.describe('signed in', () => {
    test.beforeEach(async ({ page }) => {
      await login(page);
    });

    // The daily-use surfaces, plus one dense financial table. Each waits for its
    // own content rather than a blanket timeout, so a scan never runs against a
    // page that is still skeletons.
    const routes: [name: string, path: (orgId: number) => string][] = [
      ['dashboard', (id) => `/organizations/${id}/dashboard`],
      ['children', (id) => `/organizations/${id}/children`],
      ['employees', (id) => `/organizations/${id}/employees`],
      ['attendance', (id) => `/organizations/${id}/attendance`],
      ['sections', (id) => `/organizations/${id}/sections`],
      ['budget items', (id) => `/organizations/${id}/budget-items`],
      ['statistics overview', (id) => `/organizations/${id}/statistics`],
    ];

    for (const [name, path] of routes) {
      test(`the ${name} page has no violations`, async ({ page }) => {
        const org = await getFirstOrganization(page);
        await page.goto(path(org.id));
        await page.waitForLoadState('load');
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15000 });

        const results = await scan(page);
        expect(results.violations, format(results)).toEqual([]);
      });
    }

    test('an open form dialog has no violations', async ({ page }) => {
      // A dialog is a different a11y surface from the page under it: it takes
      // the focus, it names itself, and it is where nearly every input in the
      // app lives. Scanning only closed pages would miss all of that.
      const org = await getFirstOrganization(page);
      await page.goto(`/organizations/${org.id}/children`);
      await page.waitForLoadState('load');

      await page
        .getByRole('button', { name: /new child|add child/i })
        .first()
        .click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible({ timeout: 10000 });

      const results = await scan(page);
      expect(results.violations, format(results)).toEqual([]);
    });

    test('the mobile navigation drawer has no violations', async ({ page }) => {
      // Below md only: at wider widths there is a docked rail and no drawer to
      // open. The condition is the viewport, not whether the hamburger happens
      // to be on screen yet -- probing visibility raced hydration and skipped
      // this test under *every* project, including the phone, which made it
      // look green while covering nothing.
      const viewport = page.viewportSize();
      test.skip(!viewport || viewport.width >= 768, 'the drawer only exists below md');

      const org = await getFirstOrganization(page);
      await page.goto(`/organizations/${org.id}/dashboard`);
      await page.waitForLoadState('load');

      // Asserted, not probed: below md this button must be there.
      const hamburger = page.getByRole('button', { name: /open menu/i });
      await expect(hamburger).toBeVisible({ timeout: 15000 });
      await hamburger.click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 10000 });

      const results = await scan(page);
      expect(results.violations, format(results)).toEqual([]);
    });
  });
});
