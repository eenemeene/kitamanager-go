import { test, expect, type Page } from '@playwright/test';
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
 * It runs over every dialog that was converted, not just the first one. The
 * layout lives in three shared components — the child dialog, CrudFormDialog
 * behind seven pages, and PersonFormDialog — plus the organizations page, which
 * builds its own. Each was converted by hand, so each can be got wrong on its
 * own; visual baselines were the only thing covering three of the four, and a
 * baseline cannot tell whether Save is still pressable.
 *
 * Runs in every project: the phone is where a dialog is most likely to scroll,
 * but the structure has to hold on a tablet and a desktop too.
 */
test.use({ locale: 'en-US' });

interface DialogUnderTest {
  /** What the user calls it, for the test name. */
  name: string;
  /** Which component owns the layout, so a failure names the file to open. */
  component: string;
  open: (page: Page) => Promise<void>;
}

const dialogs: DialogUnderTest[] = [
  {
    name: 'create child',
    component: 'child-create-dialog.tsx',
    open: async (page) => {
      await page.goto('/organizations/1/children');
      await page.waitForLoadState('load');
      await page
        .getByRole('button', { name: /new child|add child/i })
        .first()
        .click();
    },
  },
  {
    name: 'create section',
    component: 'crud-form-dialog.tsx',
    open: async (page) => {
      await page.goto('/organizations/1/sections');
      await page.waitForLoadState('load');
      // The create button lives behind the Manage tab; force, because the tab
      // strip animates in and Playwright otherwise races it.
      await page.getByRole('tab', { name: /manage/i }).click({ force: true });
      await page.getByRole('button', { name: /new section/i }).click();
    },
  },
  {
    name: 'create employee',
    component: 'person-form-dialog.tsx',
    open: async (page) => {
      await page.goto('/organizations/1/employees');
      await page.waitForLoadState('load');
      await page.getByRole('button', { name: /new employee/i }).click();
    },
  },
  {
    name: 'create organization',
    component: 'organizations/page.tsx',
    open: async (page) => {
      await page.goto('/organizations');
      await page.waitForLoadState('load');
      await page.getByRole('button', { name: /new organization/i }).click();
    },
  },
];

/**
 * Opens the dialog and submits it empty, which every one of these forms rejects.
 * Returns once the rejection is on screen, because that is the state the layout
 * has to survive — the content grows, and the footer must not move with it.
 */
async function openAndReject(page: Page, dialog: DialogUnderTest) {
  await login(page);
  await dialog.open(page);

  const d = page.getByRole('dialog');
  await expect(d).toBeVisible();

  const save = d.getByRole('button', { name: /^(create|save|add)/i }).first();
  const before = await save.boundingBox();
  await save.click();

  // Not every one of these forms renders the summary yet, so the signal is
  // whichever of the two arrived: a summary, or a marked input.
  await expect(
    d.locator('[data-testid="form-error-summary"], [aria-invalid="true"]').first()
  ).toBeVisible();

  return { dialog: d, save, before };
}

for (const under of dialogs) {
  test.describe(`${under.name} dialog (${under.component})`, () => {
    test('the submit button sits outside the scrolling area', async ({ page }) => {
      const { dialog, save } = await openAndReject(page, under);

      // The structural half of the guarantee, and the only half a short dialog
      // can express: with the footer inside the scrolling body, Save moves with
      // the content, and whether it moves *off screen* then depends on how many
      // fields the form happens to have. Asserting the shape instead holds for
      // every dialog, including the ones that fit today and grow a field later.
      const insideScrollArea = await save.evaluate(
        (el) => el.closest('[data-dialog-body]') !== null
      );
      expect(
        insideScrollArea,
        'the footer must be a sibling of the scrolling body, not inside it'
      ).toBe(false);

      // And the body really is the scroll container, so "outside it" means
      // something — a body that does not scroll would make the whole dialog
      // scroll instead, taking the footer with it.
      const body = dialog.locator('[data-dialog-body]');
      await expect(body).toHaveCount(1);
      const overflow = await body.evaluate((el) => getComputedStyle(el).overflowY);
      expect(overflow, 'the dialog body is what scrolls').toMatch(/auto|scroll/);
    });

    test('the submit button stays reachable when errors appear', async ({ page }) => {
      const { dialog, save, before } = await openAndReject(page, under);

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
      await openAndReject(page, under);

      const unreachable = await page.evaluate(() => {
        const d = document.querySelector('[role="dialog"]') as HTMLElement;
        const bad: string[] = [];
        d.querySelectorAll<HTMLElement>(
          'input, button, [role="checkbox"], select, textarea'
        ).forEach((el) => {
          // Skip what is not meant to be interactive. Radix Select renders a
          // hidden native <select> beside its button for form compatibility: it
          // is aria-hidden with pointer-events disabled, so "not hit-testable"
          // is correct for it and says nothing about the layout.
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
        });
        return bad;
      });

      expect(unreachable, 'every control in a dialog must be clickable').toEqual([]);
    });
  });
}
