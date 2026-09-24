/**
 * Kita-year arithmetic.
 *
 * A Kita year runs 1 August – 31 July. This module is the single definition of
 * that boundary for anything that groups, labels or windows data by it; the
 * chart helpers re-export `kitaYearLabel` so existing imports keep working.
 */

/** Returns the Kita year label for a date. August 2024 → "24/25". */
export function kitaYearLabel(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  const month = date.getMonth(); // 0-indexed
  const year = date.getFullYear();
  const startYear = month >= 7 ? year : year - 1; // Aug (7) starts a new Kita year
  const sy = String(startYear).slice(2);
  const ey = String(startYear + 1).slice(2);
  return `${sy}/${ey}`;
}

/** One request window for the bill-vs-contract comparison. */
export interface KitaYearCompareWindow {
  /** Kita year label, e.g. "25/26". Also the key its summary is stored under. */
  kitaYear: string;
  /** Earliest billed month in this Kita year, YYYY-MM-01, inclusive. */
  from: string;
  /** Latest billed month in this Kita year, YYYY-MM-01, inclusive. */
  to: string;
}

/**
 * Groups billed months into one comparison window per Kita year.
 *
 * The windows used to be 12-month blocks counted from the first billed month,
 * which only lines up with a Kita year when that month happens to be an August.
 * Every other case produced windows straddling two Kita years, and the summary
 * a year's row displayed was then chosen by "which window overlaps most" — so a
 * year's deficit analysis could be computed over months belonging to the next
 * one. Cutting on the Kita-year boundary instead makes window and year the same
 * thing, and the lookup an equality rather than a heuristic.
 *
 * Each window is clamped to the months actually billed, so it never asks the
 * API for a range with no bills in it and never exceeds twelve months.
 *
 * Input need not be sorted or unique. Months that are empty strings are
 * ignored. Returns windows ordered by `from`.
 */
export function buildKitaYearCompareWindows(billMonths: string[]): KitaYearCompareWindow[] {
  const byYear = new Map<string, string[]>();
  for (const month of billMonths) {
    if (!month) continue;
    const label = kitaYearLabel(month);
    const months = byYear.get(label);
    if (months) {
      months.push(month);
    } else {
      byYear.set(label, [month]);
    }
  }

  const windows: KitaYearCompareWindow[] = [];
  for (const [kitaYear, months] of byYear) {
    // Lexicographic sort is chronological for YYYY-MM-DD.
    const sorted = [...months].sort();
    windows.push({ kitaYear, from: sorted[0]!, to: sorted[sorted.length - 1]! });
  }
  windows.sort((a, b) => (a.from < b.from ? -1 : a.from > b.from ? 1 : 0));
  return windows;
}
