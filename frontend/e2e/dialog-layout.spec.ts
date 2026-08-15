import { test, expect } from '@playwright/test';
import { login } from './utils/test-helpers';

/**
 * A converted dialog keeps its actions reachable and covers nothing.
 *
 * Both halves are regressions that actually shipped in this branch before being
 * caught. The footer used to scroll with the content, so showing the error
 * summary pushed Save below the fold — press Save, and Save is gone. Pinning it
 * with `sticky` fixed that and covered the control directly above it, which
 * Playwright reported as "intercepts pointer events" and a user would experience
 * as a checkbox that does nothing.
 *
 * The dialog is now three regions — header, scrolling body, footer — so neither
 * is expressible. Below `sm` it is the whole screen, which removes the competing
 * scroll containers entirely.
 *
 * Runs in every project: the phone is where a dialog is most likely to scroll,
 * but the structure has to hold on a tablet and a desktop too.
 */
test.use({ locale: 'en-US' });

async function openChildDialog(page: import('@playwright/test').Page) {
  await login(page);
  await page.goto('/organizations/1/children');
  await page.waitForLoadState('load');
  await page
    .getByRole('button', { name: /new child|add child/i })
    .first()
    .click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  return dialog;
}

test('the submit button stays reachable when errors appear', async ({ page }) => {
  const dialog = await openChildDialog(page);
  const save = dialog.getByRole('button', { name: /^(create|save|add)/i }).first();

  const before = await save.boundingBox();
  await save.click();
  await expect(dialog.getByTestId('form-error-summary')).toBeVisible();
  const after = await save.boundingBox();
  const box = await dialog.boundingBox();

  expect(before, 'the submit button should be laid out before submitting').not.toBeNull();
  expect(after, 'and after').not.toBeNull();
  // Inside the dialog, and still on screen — the footer is a sibling of the
  // scrolling body, so growing the content cannot push it away.
  expect(after!.y + after!.height).toBeLessThanOrEqual(box!.y + box!.height + 1);
  await expect(save).toBeInViewport();
});

test('no control is covered or clipped by the dialog chrome', async ({ page }) => {
  const dialog = await openChildDialog(page);
  await dialog
    .getByRole('button', { name: /^(create|save|add)/i })
    .first()
    .click();
  await expect(dialog.getByTestId('form-error-summary')).toBeVisible();

  const unreachable = await page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]') as HTMLElement;
    const bad: string[] = [];
    d.querySelectorAll<HTMLElement>('input, button, [role="checkbox"], select, textarea').forEach(
      (el) => {
        // Skip what is not meant to be interactive. Radix Select renders a
        // hidden native <select> beside its button for form compatibility: it is
        // aria-hidden with pointer-events disabled, so "not hit-testable" is
        // correct for it and says nothing about the layout.
        if (el.closest('[aria-hidden="true"]')) return;
        const style = getComputedStyle(el);
        if (
          style.pointerEvents === 'none' ||
          style.visibility === 'hidden' ||
          Number(style.opacity) === 0
        ) {
          return;
        }
        el.scrollIntoView({ block: 'nearest' });
        const r = el.getBoundingClientRect();
        if (r.width === 0 || r.height === 0) return;
        const top = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
        // A null hit test means the point is outside the viewport or clipped —
        // an earlier version of this test skipped that case and could not fail
        // at all, which is how a dialog that clipped its own footer passed it.
        if (top === null) {
          bad.push(`${el.id || el.tagName} is not hit-testable`);
        } else if (top !== el && !el.contains(top) && !top.contains(el)) {
          bad.push(`${el.id || el.tagName} is under ${top.tagName.toLowerCase()}`);
        }
      }
    );
    return bad;
  });

  expect(unreachable, 'every control in a dialog must be clickable').toEqual([]);
});
