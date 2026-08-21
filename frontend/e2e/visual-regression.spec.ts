import { test, expect, type Page, type Locator } from '@playwright/test';
import { login, getFirstOrganization, resetBerlinFundingFromConfig } from './utils/test-helpers';

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

test.describe('Visual Regression - Login', () => {
  test('login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('load');

    // Login page has no dashboard chrome (no sidebar version footer)
    // and no dynamic data, but applying dynamicMasks here is a no-op
    // safety net in case future copy adds a build-hash banner.
    await expect(page).toHaveScreenshot('login-page.png', {
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

    await expect(page).toHaveScreenshot('organizations-list.png', {
      maxDiffPixelRatio: 0.02,
      mask: dynamicMasks(page),
    });
  });

  test('employees list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');
    // Wait for table to render
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('employees-list.png', {
      maxDiffPixelRatio: 0.03,
      mask: dynamicMasks(page),
    });
  });

  test('children list', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');
    await expect(page.locator('table, [role="table"]').first()).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('children-list.png', {
      maxDiffPixelRatio: 0.03,
      mask: dynamicMasks(page),
    });
  });

  test('sections board', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('load');
    // Wait for the kanban board to render
    await expect(page.getByText(/drag children/i)).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('sections-board.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('statistics overview page', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/statistics`);
    // Wait for statistics cards to render (avoid networkidle — react-query background requests prevent it)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10000 });

    await expect(page).toHaveScreenshot('statistics-overview.png', {
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
    await expect(page).toHaveScreenshot('statistics-financials.png', {
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

    await expect(page).toHaveScreenshot('create-organization-dialog.png', {
      maxDiffPixelRatio: 0.01,
      mask: dynamicMasks(page),
    });
  });

  test('create employee dialog', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');

    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await expect(page).toHaveScreenshot('create-employee-dialog.png', {
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
    await expect(dialog).toHaveScreenshot('create-child-dialog.png', {
      maxDiffPixelRatio: 0.02,
    });
  });
});
