import { test, expect, type Page } from '@playwright/test';
import { login, createTestOrg, deleteTestOrg } from './utils/test-helpers';

// MonthStepper is the date-picker shared by the Employees and
// Children list pages. The unit tests in
// frontend/src/components/ui/__tests__/month-stepper.test.tsx cover
// the click semantics; this E2E proves the year-navigation chevrons
// and the calendar dropdown survive the real Next.js render +
// react-day-picker integration.
//
// English locale on purpose so the aria-labels and dropdown headings
// match — the component reads `t('previousYear')` etc. from the
// `common` i18n namespace.
test.use({ locale: 'en-US' });

// The button shows the date in "d. MMMM yyyy" format (e.g.
// "1. January 2026"). Extract the year as a Date for assertion math
// — per CLAUDE.md / feedback_date_objects_never_strings, date logic
// runs on Date objects, not regex over the formatted string. The
// label parses cleanly through Date because the format is
// monotonic in year position (last token is always the four-digit
// year), so we read it back via the same shape that produced it.
async function getActiveOnDate(page: Page): Promise<Date> {
  // Read the value, not the label. The label is localized and, below `sm`,
  // deliberately abbreviated so the stepper fits a 412px row — so parsing it
  // made this test depend on both the locale and the viewport. It broke exactly
  // that way when the short form landed.
  const raw = await page.getByTestId('month-stepper-value').first().getAttribute('data-value');
  if (!raw) throw new Error('MonthStepper exposes no data-value');
  const parsed = new Date(`${raw}T00:00:00`);
  if (isNaN(parsed.getTime())) throw new Error(`could not parse data-value as Date: ${raw}`);
  return parsed;
}

test.describe('MonthStepper year navigation', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'MonthStepperE2E');
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
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');
    // Wait for the picker button to mount so the label is readable.
    await expect(page.getByRole('button', { name: /previousYear|Previous year/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('previousYear chevron moves the active-on date back exactly one year', async ({ page }) => {
    const before = await getActiveOnDate(page);

    await page.getByRole('button', { name: /^Previous year$/i }).click();

    // Re-read the label and assert the year decreased by 1, month
    // preserved. We compute the expected year from `before` rather
    // than hard-coding so the test stays valid as the calendar's
    // "today" drifts forward.
    const after = await getActiveOnDate(page);
    expect(after.getFullYear()).toBe(before.getFullYear() - 1);
    expect(after.getMonth()).toBe(before.getMonth());
  });

  test('nextYear chevron moves the active-on date forward exactly one year', async ({ page }) => {
    const before = await getActiveOnDate(page);

    await page.getByRole('button', { name: /^Next year$/i }).click();

    const after = await getActiveOnDate(page);
    expect(after.getFullYear()).toBe(before.getFullYear() + 1);
    expect(after.getMonth()).toBe(before.getMonth());
  });

  test('calendar popover exposes a year dropdown for direct jumps', async ({ page }) => {
    // Open the popover by clicking the date label button.
    await page.getByTestId('month-stepper-value').first().click();

    // react-day-picker renders the year selector as a native
    // <select> when captionLayout="dropdown" is on. The accessible
    // name is "Year" in English. Use getByRole to dodge styling
    // changes and to confirm the element is actually a focusable
    // select (a div with role=combobox would also pass a generic
    // text search but wouldn't satisfy the user's "select another
    // year" requirement).
    const yearSelect = page.getByRole('combobox', { name: /year/i });
    await expect(yearSelect).toBeVisible({ timeout: 5000 });

    // Capture the current year, pick the option one year ahead,
    // and assert the picker advanced. The popover's month grid
    // re-renders on dropdown change; we close it implicitly by
    // selecting a date.
    const before = await getActiveOnDate(page);
    const targetYear = String(before.getFullYear() + 2);

    // Native <select> path: selectOption by label.
    await yearSelect.selectOption(targetYear);

    // The popover caption should now read the new year. Pick any
    // visible day to commit the selection and close the popover.
    // The day "15" is in every month, so it's a stable click target.
    await page.getByRole('gridcell').getByRole('button', { name: '15' }).first().click();

    const after = await getActiveOnDate(page);
    expect(after.getFullYear()).toBe(before.getFullYear() + 2);
  });
});
