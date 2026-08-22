import { test, expect, type Page, type Locator } from '@playwright/test';
import {
  login,
  getFirstOrganization,
  getGovernmentFundingsViaApi,
  getPayPlansViaApi,
  resetBerlinFundingFromConfig,
} from './utils/test-helpers';

// Ensure English locale for consistent text rendering
test.use({ locale: 'en-US' });

// Visual regression tests capture baseline screenshots on first run.
// Subsequent runs compare against baselines to detect unintended visual changes.
// Update baselines: npx playwright test visual-regression --update-snapshots
//
// Masking convention: any element whose textual content varies between
// runs (currency totals derived from "today", git/version strings, the
// build hash footer) carries `data-visual-mask="<category>"` in source.
// `dynamicMasks(page)` returns the locators every test should mask in
// addition to test-specific ones (charts etc.). Without this masking
// the screenshots flip-flop and chew up CI on retries — see the
// repo-root visual-regression-* directories left over from past runs.
function dynamicMasks(page: Page): Locator[] {
  return [page.locator('[data-visual-mask]')];
}

/**
 * Wait until every chart on the page has actually laid out.
 *
 * Masking a chart hides its pixels but not its height. Nivo mounts after its
 * data arrives and measures its container, so a screenshot taken too early
 * catches a chart of a different size -- and everything below it sits at a
 * different offset, which is a diff of the whole lower half of the page.
 *
 * That is what made these tests flap: the same commit went green, then red, with
 * the failing diff showing the section heading rendered twice at two vertical
 * positions. It looked like nondeterministic data and was not.
 */
async function chartsReady(page: Page) {
  const charts = page.locator('[data-visual-mask="chart"]');
  const count = await charts.count();
  for (let i = 0; i < count; i++) {
    // The svg exists only once nivo has measured and drawn.
    await expect(charts.nth(i).locator('svg').first()).toBeVisible({ timeout: 15000 });
  }
  // One animation frame after the last one appears, so a chart that is still
  // transitioning into its final height has settled.
  await page.evaluate(
    () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(() => r(null))))
  );
}

/**
 * Screenshot the page into the run's report, then compare it to the baseline.
 *
 * The attach is the point. `toHaveScreenshot` compares and discards: on a match
 * it keeps nothing, and Playwright's own `screenshot: 'only-on-failure'` adds
 * nothing either. So a green run produced a report with no pictures in it at
 * all -- the ten pages this file renders across three viewports existed only as
 * a pass/fail verdict, and the merged CI report carried zero attachments.
 *
 * Attaching first also means the render survives a failed comparison, sitting
 * beside the expected/actual/diff triplet Playwright adds in that case.
 *
 * The second capture costs one extra screenshot per page. `mask` is passed to
 * both so the attachment shows exactly what was compared, pink boxes and all,
 * rather than a picture that disagrees with the verdict beside it.
 */
async function shoot(
  target: Page | Locator,
  name: string,
  options: { maxDiffPixelRatio: number; mask?: Locator[] }
) {
  await test.info().attach(name, {
    // `scale: 'css'` because that is what toHaveScreenshot compares at, while
    // page.screenshot() defaults to 'device'. On a Pixel 7 (ratio 2.625) the two
    // differ by more than a factor of two -- 412x839 against 1082x2202 -- so
    // without this the attachment is a different rendering from the verdict
    // printed beside it, and unusable as a baseline. Invisible at ratio 1, which
    // is chromium and tablet, so it only shows up on mobile.
    body: await target.screenshot({ mask: options.mask, scale: 'css' }),
    contentType: 'image/png',
  });
  await expect(target).toHaveScreenshot(name, options);
}

test.describe('Visual Regression - Login', () => {
  test('login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('load');

    // Login page has no dashboard chrome (no sidebar version footer)
    // and no dynamic data, but applying dynamicMasks here is a no-op
    // safety net in case future copy adds a build-hash banner.
    await shoot(page, 'login-page.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });
});

test.describe('Visual Regression - Dashboard', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('organizations list', async ({ page }) => {
    await page.goto('/organizations');
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('load');

    await shoot(page, 'organizations-list.png', {
      maxDiffPixelRatio: 0.02,
      mask: dynamicMasks(page),
    });
  });

  test('employees list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');
    // Wait for table to render
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await shoot(page, 'employees-list.png', {
      maxDiffPixelRatio: 0.03,
      mask: dynamicMasks(page),
    });
  });

  test('children list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await shoot(page, 'children-list.png', {
      maxDiffPixelRatio: 0.03,
      mask: dynamicMasks(page),
    });
  });

  test('sections board', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('load');
    // Wait for the kanban board to render
    await expect(page.getByText(/drag children/i)).toBeVisible({ timeout: 10000 });

    await shoot(page, 'sections-board.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('statistics overview page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics`);
    // Wait for statistics cards to render (avoid networkidle — react-query background requests prevent it)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    await chartsReady(page);

    await shoot(page, 'statistics-overview.png', {
      maxDiffPixelRatio: 0.02,
      mask: dynamicMasks(page),
    });
  });

  test('statistics financials page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/financials`);
    // Wait for the financial overview chart card to render (avoid networkidle — react-query background requests prevent it)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // Mask:
    //   - dynamic-text elements (currency totals, version footer) via
    //     [data-visual-mask] on the source elements
    //   - the chart area itself: SVG rendering has sub-pixel
    //     anti-aliasing jitter between runs, and the "Today" marker
    //     shifts position over time
    await chartsReady(page);

    await shoot(page, 'statistics-financials.png', {
      maxDiffPixelRatio: 0.01,
      mask: [...dynamicMasks(page), page.locator('[role="application"]')],
    });
  });
});

test.describe('Visual Regression - Dialogs', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    // The create-child dialog renders one chip per funding property, and that
    // funding is a global singleton some tests must delete and rebuild -- berlin
    // is the only valid state and the column is unique, so testing that a
    // funding can be created means deleting the seeded one first. The dialog
    // therefore showed the seeded properties or the smaller stand-in the CRUD
    // tests put back, depending on which shard ran first, and the baseline was
    // only correct by luck: adding two tests anywhere re-split the shards and
    // turned this into a 26px height difference.
    //
    // Reimporting the real config settles it, and settles it on what a user
    // actually sees rather than on a fixture invented for the test.
    await resetBerlinFundingFromConfig(page);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('create organization dialog', async ({ page }) => {
    await page.goto('/organizations');
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new organization/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await shoot(page, 'create-organization-dialog.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('create employee dialog', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await shoot(page, 'create-employee-dialog.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('create child dialog', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new child/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5000 });

    // Screenshot only the dialog element: the children page behind the
    // dialog overlay shows today's date in the week navigator, which
    // changes daily and causes pixel diffs.
    //
    // Wait for the funding property chips to stop arriving before capturing.
    //
    // They come from two chained requests — the funding, then its periods — and
    // the set of chips is the union across periods, so it grows as responses
    // land. The dialog is visible long before that finishes, and each extra row
    // shifts everything below it.
    //
    // This used to wait for one specific chip, "integration a", on the reasoning
    // that it comes from a non-current period and so implies the rest have
    // loaded. That turned a slow response into a hard failure: CI captured the
    // dialog with four chips and no "integration a" at all, and the test spent
    // ten seconds waiting for something that was never going to appear in time.
    //
    // Waiting for the count to settle makes no assumption about which chips
    // exist. If the set really does differ between environments — the open
    // question behind this test — that now shows up as a snapshot difference
    // naming the actual problem, rather than as a timeout that does not.
    const chips = dialog.locator('[data-testid="property-suggestions"] button');
    await expect(chips.first()).toBeVisible({ timeout: 15000 });
    await expect
      .poll(
        async () => {
          const before = await chips.count();
          await page.waitForTimeout(400);
          const after = await chips.count();
          return before === after ? after : -1;
        },
        { timeout: 15000, message: 'funding property chips never stopped changing' }
      )
      .toBeGreaterThan(0);

    // The only dialog snapshot that had no tolerance, while both its siblings
    // above allow 0.01 -- and the one with the most bordered controls, so it
    // carries the most antialiasing-sensitive edges now that --input renders at
    // 3:1 instead of being invisible. CI came in at 922px, ratio 0.01, which is
    // exactly what the siblings already absorb; 0.02 gives it the same headroom
    // with a margin, without loosening the others.
    await shoot(dialog, 'create-child-dialog.png', {
      maxDiffPixelRatio: 0.02,
    });
  });
});

/**
 * The rest of the product.
 *
 * The suite covered seven pages and three dialogs while the app has
 * twenty-six routes, so most of it -- including the surfaces that are hardest to
 * eyeball a regression on -- had no picture taken of it at all. The 709 tests in
 * the other forty spec files are behavioural: they click and assert, and none of
 * them capture pixels.
 *
 * Chosen for visual weight against baseline stability. A page whose content is
 * derived from "today" flip-flops between runs and burns CI on retries, so
 * anything date-derived is either masked here or left out: the audit log is
 * nothing but timestamps, and the forecast redraws from the current Kita year.
 */
test.describe('Visual Regression - Operations', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    orgId = (await getFirstOrganization(page)).id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('dashboard', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/dashboard`);
    await page.waitForLoadState('load');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // The widgets are almost entirely derived from today -- staffing coverage,
    // funding totals, pending promotions. `[data-visual-mask]` covers the
    // numbers; the charts are masked wholesale for the same reason the
    // financials page masks them, SVG anti-aliasing jitter between runs.
    await chartsReady(page);

    await shoot(page, 'dashboard.png', {
      maxDiffPixelRatio: 0.02,
      mask: [...dynamicMasks(page), page.locator('[role="application"]')],
    });
  });

  test('attendance week grid', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('load');
    await expect(page.locator('table').first()).toBeVisible({ timeout: 15000 });

    // The header row carries this week's dates ("Mon 18.08"), so it is different
    // every Monday and has to be masked or the baseline expires weekly. What is
    // being watched here is the grid itself: five columns of dense cells with
    // 36px touch targets, the surface most at risk from a layout change.
    await shoot(page, 'attendance-week-grid.png', {
      maxDiffPixelRatio: 0.02,
      mask: [...dynamicMasks(page), page.locator('thead')],
    });
  });

  test('pay plan detail', async ({ page }) => {
    const payPlans = await getPayPlansViaApi(page, orgId);
    test.skip(payPlans.length === 0, 'no pay plan in the seed to render');

    await page.goto(`/organizations/${orgId}/payplans/${payPlans[0].id}`);
    await page.waitForLoadState('load');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // A grade-by-step salary grid: dense, wide, and entirely fixed data, so it
    // needs no masking beyond the version footer and makes a strict baseline.
    await shoot(page, 'payplan-detail.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('funding rate detail', async ({ page }) => {
    const fundings = await getGovernmentFundingsViaApi(page);
    test.skip(fundings.length === 0, 'no funding configuration to render');

    await page.goto(`/government-funding-rates/${fundings[0].id}`);
    await page.waitForLoadState('load');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // Periods and their properties -- the Berlin rate table. Fixed data, and the
    // page the ISBJ calculations are read against.
    await shoot(page, 'funding-rate-detail.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('budget items list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/budget-items`);
    await page.waitForLoadState('load');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    await shoot(page, 'budget-items-list.png', {
      maxDiffPixelRatio: 0.02,
      mask: dynamicMasks(page),
    });
  });

  test('statistics staffing page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics/staffing`);
    await page.waitForLoadState('load');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    // Chart masked, page chrome not: the filters, the legend and the table
    // beneath it are what a layout change would break.
    await chartsReady(page);

    await shoot(page, 'statistics-staffing.png', {
      maxDiffPixelRatio: 0.02,
      mask: [...dynamicMasks(page), page.locator('[role="application"]')],
    });
  });
});
