import { test, expect } from '@playwright/test';
import { login, createTestOrg } from './utils/test-helpers';

/**
 * A dialog's footer must never cover a control, and never move.
 *
 * Both halves come from real failures. The footer used to scroll with the
 * content, so showing the error summary pushed the submit button below the fold
 * — press Save, and Save was gone. Pinning it with `sticky` fixed that and broke
 * the other half: the backup-codes acknowledgement sits directly above the
 * footer, and a floating bar over it made it unclickable. Playwright reported it
 * as "intercepts pointer events", which is exactly what a user would experience
 * as "the checkbox does nothing".
 *
 * The footer is now a sibling of the scrollable body rather than its last child,
 * so neither failure is expressible. This test asserts the property rather than
 * the layout: walk every control, scroll it into view the way a browser does,
 * and ask what is actually on top of it.
 *
 * Runs in every project — the phone is where a dialog is most likely to scroll,
 * but the same structure has to hold on a tablet and a desktop.
 */
test.use({ locale: 'en-US' });

test('no control in a tall dialog is covered by the footer', async ({ page }) => {
  await login(page);
  const { orgId } = await createTestOrg(page, 'Overlap');
  await page.goto(`/organizations/${orgId}/children`);
  await page.waitForLoadState('load');
  await page
    .getByRole('button', { name: /new child|add child/i })
    .first()
    .click();
  const dialog = page.getByRole('dialog');
  await dialog.waitFor();

  // Walk every focusable control, scroll it into view the way a browser does,
  // and ask what is actually on top at its centre.
  const covered = await page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]') as HTMLElement;
    const controls = Array.from(
      d.querySelectorAll<HTMLElement>('input, button, [role="checkbox"], select, textarea')
    );
    const bad: string[] = [];
    for (const el of controls) {
      el.scrollIntoView({ block: 'nearest' });
      const r = el.getBoundingClientRect();
      if (r.width === 0 || r.height === 0) continue;
      const top = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
      if (top && top !== el && !el.contains(top) && !top.contains(el)) {
        bad.push(
          `${el.tagName.toLowerCase()}#${el.id || '(no id)'} covered by ${top.tagName.toLowerCase()}.${(top.className || '').toString().slice(0, 40)}`
        );
      }
    }
    return bad;
  });

  expect(covered, 'no control may be hidden behind the dialog footer').toEqual([]);

  // And the footer itself stays put: it is outside the scroll container, so it
  // is visible whether or not the body has been scrolled.
  const footerButton = dialog.getByRole('button', { name: /^(create|save|add)/i }).first();
  await expect(footerButton).toBeInViewport();
});
