import type { Page } from '@playwright/test';

/**
 * Viewport facts the specs need, in one place.
 *
 * The children and employees tables render their secondary row actions — contract
 * history, billing history, add contract, vouchers — with `hidden sm:inline-flex`.
 * They are therefore present on a tablet (768px) and desktop, and absent on a
 * phone, where six icon buttons would not fit beside the name and section columns.
 *
 * Two of those actions navigate to pages a spec can reach with `page.goto`, so
 * phone coverage of those flows is unaffected. The other two open dialogs mounted
 * by the button's own click handler, with no URL, so a phone-sized run genuinely
 * cannot reach them — those specs declare it with `skipWithoutRowActions` rather
 * than failing or silently passing.
 */

/** Tailwind's `sm`: the width from which the tables show their secondary actions. */
export const SM_BREAKPOINT = 640;

/** True when the table's secondary row actions are out of reach at this viewport. */
export function rowActionsHidden(page: Page): boolean {
  const vp = page.viewportSize();
  return vp !== null && vp.width < SM_BREAKPOINT;
}

/**
 * Skips a spec that needs a dialog only reachable from a secondary row action.
 *
 * The reason is passed to Playwright so it shows up in the report — a skip that
 * explains itself, rather than a test that quietly does not run. If phones ever
 * gain a path to these dialogs, the skip stops triggering on its own.
 */
export function skipWithoutRowActions(
  test: { skip(condition: boolean, reason: string): void },
  page: Page,
  action: string
): void {
  test.skip(
    rowActionsHidden(page),
    `"${action}" opens a dialog from a row action rendered only at >=${SM_BREAKPOINT}px ` +
      `(hidden sm:inline-flex), and no URL reaches that dialog, so it cannot be driven ` +
      `at this viewport`
  );
}
