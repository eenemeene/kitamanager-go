/**
 * Helpers for the print/report pages: parse a `?month=YYYY-MM` query
 * parameter and derive the date filters every API call needs to be
 * scoped to.
 *
 * Why each API call needs explicit dates: most statistics endpoints
 * default to a ~25-month "Kita year context" window when from/to are
 * omitted — fine for an interactive dashboard exploring several years
 * at once, wrong for a printed report representing a specific month.
 * If a print page omits the dates, the rendered chart silently spans
 * a different period than the page title claims.
 *
 * Three shapes of date filter the print pages need, depending on
 * what the chart/table is meant to show:
 *
 *   1. Snapshots ("the children active right now"): use `asOf` —
 *      the first day of the report month, passed as `?date=`.
 *
 *   2. Annual matrix / grid views ("the Kita year"): use the
 *      `kitaYearFrom` / `kitaYearTo` pair (Aug 1 → Jul 1 of the Kita
 *      year that contains the report month). Kitas plan and report
 *      against the Kita year, not the calendar year.
 *
 *   3. Multi-year trends ("show me the shape over time"): use the
 *      `trendFrom` / `trendTo` pair, which spans the previous,
 *      current, and next Kita year (3 years = 36 months).
 */

export interface ReportMonth {
  /** YYYY-MM, e.g. "2026-04". */
  month: string;
  /** YYYY-MM-DD: first day of the report month. Use for `?date=` (snapshot) queries. */
  asOf: string;
  /**
   * YYYY-MM-DD: first day (Aug 1) of the Kita year containing the report month.
   * The Kita year runs Aug 1 → Jul 31. Use as `?from=` for annual matrix /
   * grid queries — that's the natural reporting cycle for German Kitas, not
   * the calendar year.
   */
  kitaYearFrom: string;
  /**
   * YYYY-MM-DD: first day (Jul 1) of the Kita year's last month. Use as `?to=`.
   * The API snaps to first-of-month and treats `to` as inclusive, so this is
   * Jul of (kitaYearFrom's year + 1) and produces a 12-row matrix Aug…Jul.
   */
  kitaYearTo: string;
  /**
   * YYYY-MM-DD: first day (Aug 1) of the *previous* Kita year. Use as `?from=`
   * for multi-year trend charts that should show prev + current + next Kita
   * years together. Spans 36 months total when paired with trendTo.
   */
  trendFrom: string;
  /**
   * YYYY-MM-DD: first day (Jul 1) of the last month of the *next* Kita year.
   * Use as `?to=` for trend queries.
   */
  trendTo: string;
  /** Year as integer for headings (the calendar year of the report month). */
  year: number;
  /** 1-12 month number. */
  monthNumber: number;
  /** Calendar year of `kitaYearFrom`. Used for headings like "Aug 2025 – Jul 2026". */
  kitaYearStartYear: number;
}

const pad2 = (n: number) => n.toString().padStart(2, '0');

/**
 * Parse a YYYY-MM string (typically from `?month=`) into the derived
 * dates a print page needs. Falls back to the current calendar month
 * when the input is missing or malformed — the latter so a developer
 * hitting the URL manually without ?month doesn't see an error page,
 * and so a typo (?month=2026/04) renders something rather than nothing.
 */
export function parseReportMonth(input: string | null | undefined): ReportMonth {
  let year: number;
  let month: number; // 1-12

  if (input && /^\d{4}-\d{2}$/.test(input)) {
    const [yStr, mStr] = input.split('-');
    year = parseInt(yStr, 10);
    month = parseInt(mStr, 10);
    if (year < 2000 || year > 2100 || month < 1 || month > 12) {
      ({ year, month } = currentYearMonth());
    }
  } else {
    ({ year, month } = currentYearMonth());
  }

  const monthStr = `${year}-${pad2(month)}`;
  const asOf = `${year}-${pad2(month)}-01`;

  // Kita year containing the report month: starts Aug 1 of (year)
  // when month >= August, otherwise Aug 1 of (year - 1). Mirrors the
  // backend's snapDateRange logic so the windows we send the API line
  // up with what its own defaults would compute for the same period.
  const kitaYearStartYear = month >= 8 ? year : year - 1;

  // Date.UTC handles month under/overflow correctly when years roll over.
  // Day=1 throughout so leap-year edge cases never come up.
  const kitaStart = new Date(Date.UTC(kitaYearStartYear, 7, 1)); // Aug = month 7 (0-indexed)
  const kitaEnd = new Date(Date.UTC(kitaYearStartYear + 1, 6, 1)); // Jul of next year
  const trendStart = new Date(Date.UTC(kitaYearStartYear - 1, 7, 1));
  const trendEnd = new Date(Date.UTC(kitaYearStartYear + 2, 6, 1));

  return {
    month: monthStr,
    asOf,
    kitaYearFrom: formatYYYYMMDD(kitaStart),
    kitaYearTo: formatYYYYMMDD(kitaEnd),
    trendFrom: formatYYYYMMDD(trendStart),
    trendTo: formatYYYYMMDD(trendEnd),
    year,
    monthNumber: month,
    kitaYearStartYear,
  };
}

function currentYearMonth(): { year: number; month: number } {
  const now = new Date();
  return { year: now.getFullYear(), month: now.getMonth() + 1 };
}

function formatYYYYMMDD(d: Date): string {
  return `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}`;
}

/**
 * Format a ReportMonth as a long human-readable label, e.g. "April 2026"
 * (or "abril de 2026" depending on locale). Used for print-page subtitles.
 * Locale defaults to the runtime default — matching the rest of the print
 * pages' use of `toLocaleDateString()`.
 */
export function formatReportMonthLong(rm: ReportMonth, locale?: string): string {
  const date = new Date(Date.UTC(rm.year, rm.monthNumber - 1, 1));
  return date.toLocaleDateString(locale, { month: 'long', year: 'numeric', timeZone: 'UTC' });
}

/**
 * Format the Kita-year range a ReportMonth covers as e.g. "Aug 2025 – Jul 2026".
 * Used as a subtitle on annual matrix / grid sections so a reader can see at
 * a glance which 12 months the table represents (and why it's not aligned
 * with the calendar year).
 */
export function formatKitaYearLabel(rm: ReportMonth, locale?: string): string {
  const start = new Date(Date.UTC(rm.kitaYearStartYear, 7, 1));
  const end = new Date(Date.UTC(rm.kitaYearStartYear + 1, 6, 1));
  const startLabel = start.toLocaleDateString(locale, {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  });
  const endLabel = end.toLocaleDateString(locale, {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  });
  return `${startLabel} – ${endLabel}`;
}
