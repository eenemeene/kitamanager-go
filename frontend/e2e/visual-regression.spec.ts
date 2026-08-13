import { test, expect, type Page, type Locator } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

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
    // Wait for the funding property chips before capturing. They arrive after two
    // chained requests — the funding list, then that funding's periods — so the
    // dialog is visible well before its lower half is complete. Screenshotting on
    // visibility alone captured whichever moment the run happened to reach: this
    // snapshot has been observed with zero, four and eight chips, and each extra
    // row of chips shifts everything below it. That is what the old
    // `maxDiffPixelRatio` was absorbing, and why the tablet copy failed in CI
    // while passing locally.
    // Wait for the funding property chips to arrive before capturing: they come
    // from two chained requests (the funding list, then that funding's periods),
    // so the dialog is visible well before its lower half is complete. Without
    // this the snapshot caught whichever moment the run reached — zero chips has
    // been observed as often as a full row.
    await expect(dialog.getByText(/Available:/i)).toBeVisible({ timeout: 10000 });

    // The chips themselves are masked, and not because pixels are inconvenient:
    // which properties an organization offers comes from its government funding
    // configuration, and that set differs between this checkout and CI even with
    // an identical berlin.yaml — four available chips here, eight there. A
    // baseline generated in one environment is therefore wrong in the other by
    // construction, which is exactly what made the tablet copy of this snapshot
    // the single failure of the first CI run.
    //
    // Masking that one field excludes an input the test does not control while
    // keeping everything this file exists to protect — the dialog's width,
    // padding and field layout — under exact comparison. The alternative, the
    // diff ratio this replaces, silently tolerated real layout drift everywhere
    // else in the dialog.
    await expect(dialog).toHaveScreenshot('create-child-dialog.png', {
      mask: [dialog.locator('[data-testid="contract-properties-field"]')],
    });
  });
});
