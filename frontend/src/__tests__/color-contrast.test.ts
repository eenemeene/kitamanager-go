/**
 * The design tokens have to clear WCAG 2.2 AA, and nothing else checks that.
 *
 * The light palette shipped with `--success` at 2.24:1, `--warning` at 2.09:1
 * and `--destructive` at 3.69:1 against the page background — the last of those
 * is the colour every validation message in the app is painted in. `--input`,
 * which draws the boundary of every text field, was at 1.25:1 in both themes.
 * None of it was visible from the code: an HSL triple in a stylesheet does not
 * announce its contrast ratio, so the values drifted below the threshold and
 * stayed there.
 *
 * This reads the real tokens out of globals.css rather than restating them, so
 * editing a colour is what runs the check. The pairs below are the ones that
 * actually meet on screen, including a token used as text on a tint of itself
 * (`bg-warning/15 text-warning`) — that composite is the tightest constraint in
 * the light theme and is what pins those tokens as dark as they are.
 */

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

type Hsl = { h: number; s: number; l: number };
type Rgb = [number, number, number];

const CSS = readFileSync(join(__dirname, '..', 'app', 'globals.css'), 'utf8');

/** Pulls one `--token: H S% L%;` declaration out of a `:root` / `.dark` block. */
function parseTokens(selector: string): Record<string, Hsl> {
  const block = new RegExp(`${selector}\\s*\\{([\\s\\S]*?)\\n\\s*\\}`).exec(CSS);
  if (!block) {
    throw new Error(`no ${selector} block in globals.css`);
  }
  const tokens: Record<string, Hsl> = {};
  const decl = /--([a-z-]+):\s*([\d.]+)\s+([\d.]+)%\s+([\d.]+)%\s*;/g;
  let m: RegExpExecArray | null;
  while ((m = decl.exec(block[1])) !== null) {
    tokens[m[1]] = { h: Number(m[2]), s: Number(m[3]), l: Number(m[4]) };
  }
  return tokens;
}

function hslToRgb({ h, s, l }: Hsl): Rgb {
  const sat = s / 100;
  const lig = l / 100;
  const c = (1 - Math.abs(2 * lig - 1)) * sat;
  const hp = h / 60;
  const x = c * (1 - Math.abs((hp % 2) - 1));
  const [r, g, b] = [
    [c, x, 0],
    [x, c, 0],
    [0, c, x],
    [0, x, c],
    [x, 0, c],
    [c, 0, x],
  ][Math.floor(hp) % 6];
  const m = lig - c / 2;
  return [r + m, g + m, b + m];
}

/** Tailwind's `/15` opacity suffix composited over an opaque surface. */
function blend(fg: Rgb, alpha: number, bg: Rgb): Rgb {
  return [0, 1, 2].map((i) => fg[i] * alpha + bg[i] * (1 - alpha)) as Rgb;
}

function relativeLuminance([r, g, b]: Rgb): number {
  const lin = (v: number) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4);
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

function contrast(a: Rgb, b: Rgb): number {
  const [hi, lo] = [relativeLuminance(a), relativeLuminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const light = parseTokens(':root');
const dark = parseTokens('\\.dark');

/** WCAG 1.4.3 — body text. */
const AA_TEXT = 4.5;
/** WCAG 1.4.11 — the visual boundary that identifies a control. */
const AA_NON_TEXT = 3;

describe.each([
  ['light', light],
  ['dark', dark],
])('%s theme', (themeName, t) => {
  const rgb = (token: string): Rgb => {
    if (!t[token]) {
      throw new Error(`--${token} missing from the ${themeName} palette`);
    }
    return hslToRgb(t[token]);
  };

  // Every token rendered as text, against both surfaces it can sit on.
  describe.each([
    'foreground',
    'muted-foreground',
    'primary',
    'destructive',
    'success',
    'warning',
    'info',
  ])('text-%s', (token) => {
    it.each(['background', 'card'])(`meets ${AA_TEXT}:1 on --%s`, (surface) => {
      expect(contrast(rgb(token), rgb(surface))).toBeGreaterThanOrEqual(AA_TEXT);
    });
  });

  // The status-pill pattern: `bg-warning/15 text-warning` and friends. The tint
  // lifts the background toward the text colour, so this is stricter than the
  // plain-surface case above and is what actually bounds these tokens.
  describe.each(['destructive', 'success', 'warning', 'info'])(
    'text-%s on a tint of itself',
    (token) => {
      it.each([0.1, 0.15])(`meets ${AA_TEXT}:1 over bg-${'%s'}`, (alpha) => {
        for (const surface of ['background', 'card'] as const) {
          expect(
            contrast(rgb(token), blend(rgb(token), alpha, rgb(surface)))
          ).toBeGreaterThanOrEqual(AA_TEXT);
        }
      });
    }
  );

  // Solid fills that carry their own foreground: buttons and badges.
  it.each([
    ['primary', 'primary-foreground'],
    ['destructive', 'destructive-foreground'],
    ['success', 'success-foreground'],
    ['warning', 'warning-foreground'],
    ['info', 'info-foreground'],
    ['sidebar-active', 'sidebar-active-foreground'],
  ])('--%s carries --%s', (fill, fg) => {
    expect(contrast(rgb(fill), rgb(fg))).toBeGreaterThanOrEqual(AA_TEXT);
  });

  // Secondary text on a faint status wash — `bg-info/10` on the sections
  // kanban card, `bg-destructive/15` pills. The browser paints the composite,
  // so this pair is real even though neither token names the other. It is also
  // what caught --muted-foreground at 46%: fine on white, 3.95:1 on the wash.
  describe.each(['destructive', 'success', 'warning', 'info', 'muted'])(
    'text-muted-foreground on a %s wash',
    (tint) => {
      it.each([0.1, 0.15])(`meets ${AA_TEXT}:1 at /%s opacity`, (alpha) => {
        for (const surface of ['background', 'card'] as const) {
          expect(
            contrast(rgb('muted-foreground'), blend(rgb(tint), alpha, rgb(surface)))
          ).toBeGreaterThanOrEqual(AA_TEXT);
        }
      });
    }
  );

  it(`--input draws a boundary at ${AA_NON_TEXT}:1`, () => {
    for (const surface of ['background', 'card'] as const) {
      expect(contrast(rgb('input'), rgb(surface))).toBeGreaterThanOrEqual(AA_NON_TEXT);
    }
  });

  it(`--ring is visible against the page at ${AA_NON_TEXT}:1`, () => {
    expect(contrast(rgb('ring'), rgb('background'))).toBeGreaterThanOrEqual(AA_NON_TEXT);
  });
});
