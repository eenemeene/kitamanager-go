/**
 * Assertions for the accessibility properties a form is supposed to have.
 *
 * # Why there are two of these and not one
 *
 * axe is the obvious tool and it is not sufficient on its own. The bug that
 * prompted this file was `<Label htmlFor="create_gender">` pointing at an
 * element that does not exist, because `GenderSelect` accepts no `id` prop and
 * `PropertyTagInput` destructured one and never rendered it. Clicking the label
 * did nothing and the field was announced without its name.
 *
 * Run against that dialog, axe reports the gender select only as `button-name`
 * — a Radix `SelectTrigger` is a button, and a button with no text and no
 * associated label has no accessible name — and says nothing at all about the
 * properties input, which is a div. It has no rule for "this label points at
 * nothing", because in isolation an orphan `<label>` is not itself a violation.
 *
 * eslint sees even less. `jsx-a11y/label-has-associated-control` passes on the
 * dialog: syntactically the label has an `htmlFor` and there is a control
 * beside it. Whether the id survives into the DOM is a question about two
 * components at once, which a per-file lint rule cannot answer.
 *
 * So `expectNoOrphanLabels` checks the one thing neither of them does, on the
 * rendered DOM where the answer actually exists, and `expectNoA11yViolations`
 * covers the much wider set axe is good at.
 */

import { axe } from 'jest-axe';

/**
 * Every `<label for>` in the tree resolves to an element that is really there.
 *
 * The failure message names the label's text, because "create_gender" alone
 * does not tell you which field on screen is broken.
 */
export function expectNoOrphanLabels(container: HTMLElement): void {
  const orphans: string[] = [];
  container.querySelectorAll('label[for]').forEach((label) => {
    const target = label.getAttribute('for');
    if (!target) {
      return;
    }
    if (!container.querySelector(`#${CSS.escape(target)}`)) {
      orphans.push(`  htmlFor="${target}" — the label reading "${label.textContent?.trim()}"`);
    }
  });

  if (orphans.length > 0) {
    throw new Error(
      `${orphans.length} label(s) point at an element that does not exist:\n${orphans.join('\n')}\n\n` +
        'Clicking these labels does nothing and the control they name is announced ' +
        'without it. Usually the id is accepted as a prop and never rendered — check ' +
        'that the component forwards it to the element the user actually focuses.'
    );
  }
}

/**
 * Radix's focus guards are sentinel spans it parks at the edges of a portal
 * with `tabindex=0`, `opacity: 0` and `pointer-events: none`, inside a subtree
 * it has marked `aria-hidden`. axe's `aria-hidden-focus` rule flags them on
 * every dialog and popover in the app. They are Radix's own machinery for
 * keeping focus inside the layer, not something this codebase can fix, so the
 * rule is off — leaving it on would make the whole check noise.
 */
const RADIX_FALSE_POSITIVES = {
  'aria-hidden-focus': { enabled: false },
} as const;

/** No axe violations, beyond the Radix internals noted above. */
export async function expectNoA11yViolations(container: HTMLElement): Promise<void> {
  const results = await axe(container, { rules: RADIX_FALSE_POSITIVES });

  if (results.violations.length > 0) {
    const detail = results.violations
      .map(
        (v) =>
          `  [${v.impact}] ${v.id}: ${v.help}\n` +
          v.nodes.map((n) => `      ${n.html.slice(0, 160)}`).join('\n')
      )
      .join('\n');
    throw new Error(`${results.violations.length} accessibility violation(s):\n${detail}`);
  }
}
