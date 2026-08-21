/**
 * Display formatting.
 *
 * Every function here that renders a number, a date or an amount takes the
 * reader's locale as a required argument. It used to be optional, and the
 * defaults disagreed with each other — `formatCurrency` defaulted to German
 * while `formatDate` defaulted to English — so a German contract card printed
 * a German euro amount beside an English date, and neither language was
 * internally consistent. 60 of 64 currency call sites and 24 of 29 date call
 * sites simply took the default.
 *
 * Requiring the argument turns that from something a caller has to remember
 * into something the compiler will not let them forget. Components should not
 * call these directly, though: `useFormatters()` in `@/hooks/use-formatters`
 * binds the locale once from `useLocale()` and is what the UI uses. These stay
 * exported for tests and for the few callers outside a React tree.
 */

import { format, parseISO, differenceInYears, type Locale as DateFnsLocale } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import type { Locale } from '@/i18n/config';
import { todayBerlinString } from './contracts';

const dateFnsLocales: Record<Locale, DateFnsLocale> = {
  de,
  en: enUS,
};

/**
 * The BCP 47 tag `Intl` needs, from the app's short locale code.
 *
 * The app stores 'de' / 'en' because that is what next-intl matches catalogues
 * on. `Intl` accepts those, but resolves bare 'de' and 'en' to region-neutral
 * defaults, and for 'en' that is not the US convention the UI was written
 * against. Naming the region keeps the output stable.
 */
export function intlLocale(locale: Locale): string {
  return locale === 'de' ? 'de-DE' : 'en-US';
}

/**
 * Format a date string for display
 */
export function formatDate(dateString: string | null | undefined, locale: Locale): string {
  if (!dateString) return '-';
  try {
    const date = parseISO(dateString);
    return format(date, 'PP', { locale: dateFnsLocales[locale] });
  } catch {
    return dateString;
  }
}

/**
 * Format a date as a month and year, the shape chart axes and table headers
 * use — "Mär 26" by default, "März 2026" with `{ month: 'long', year:
 * 'numeric' }`.
 *
 * Six chart and table files each had their own copy of this, and they did not
 * agree: the charts hardcoded 'en-US' while the tables beside them hardcoded
 * 'de-DE', so one page could label the same month "Mar 26" above "Mär 26".
 */
export function formatMonthYear(
  date: string | Date,
  locale: Locale,
  options: Intl.DateTimeFormatOptions = { month: 'short', year: '2-digit' }
): string {
  // A bare "YYYY-MM-DD" parses as UTC midnight, which is the previous day in
  // any timezone behind UTC; anchoring at local midnight keeps the month right.
  const value = typeof date === 'string' ? new Date(`${date.slice(0, 10)}T00:00:00`) : date;
  if (isNaN(value.getTime())) return typeof date === 'string' ? date : '';
  return value.toLocaleDateString(intlLocale(locale), options);
}

/** Format a number with the locale's separators. */
export function formatNumber(
  value: number,
  locale: Locale,
  options?: Intl.NumberFormatOptions
): string {
  return value.toLocaleString(intlLocale(locale), options);
}

/**
 * Format a date string for input fields (YYYY-MM-DD)
 */
export function formatDateForInput(dateString: string | null | undefined): string {
  if (!dateString) return '';
  try {
    const date = parseISO(dateString);
    return format(date, 'yyyy-MM-dd');
  } catch {
    return '';
  }
}

/**
 * Format a date string for API submission (RFC3339 format)
 * Converts "2025-01-15" to "2025-01-15T00:00:00Z"
 */
export function formatDateForApi(dateString: string | null | undefined): string | null {
  if (!dateString) return null;
  try {
    // If already in RFC3339 format, return as-is
    if (dateString.includes('T')) return dateString;
    // Convert YYYY-MM-DD to RFC3339
    return `${dateString}T00:00:00Z`;
  } catch {
    return null;
  }
}

/**
 * Calculate age from birthdate
 */
export function calculateAge(birthdate: string): number {
  try {
    const birth = parseISO(birthdate);
    if (isNaN(birth.getTime())) {
      return 0;
    }
    const age = differenceInYears(new Date(), birth);
    return isNaN(age) ? 0 : age;
  } catch {
    return 0;
  }
}

/**
 * Format currency from cents to display format
 * All monetary values from API are in cents
 */
export function formatCurrency(cents: number | null | undefined, locale: Locale): string {
  if (cents === null || cents === undefined) return '-';
  const euros = cents / 100;
  return new Intl.NumberFormat(intlLocale(locale), {
    style: 'currency',
    currency: 'EUR',
  }).format(euros);
}

/**
 * Convert euros to cents for API submission
 */
export function eurosToCents(euros: number): number {
  return Math.round(euros * 100);
}

/**
 * Convert cents to euros for form display
 */
export function centsToEuros(cents: number): number {
  return cents / 100;
}

/**
 * Format a period range
 */
export function formatPeriod(
  from: string,
  to: string | null | undefined,
  locale: Locale,
  ongoingText = 'ongoing'
): string {
  const fromFormatted = formatDate(from, locale);
  const toFormatted = to ? formatDate(to, locale) : ongoingText;
  return `${fromFormatted} - ${toFormatted}`;
}

/**
 * Format FTE (Full Time Equivalent) / staffing ratio.
 *
 * Two decimals, through the locale rather than `toFixed`, which always emits a
 * decimal point and so showed a German reader "1.50" where every other number
 * on the same row said "1,50".
 */
export function formatFte(ratio: number, locale: Locale): string {
  return formatNumber(ratio, locale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

/**
 * Format a percentage. Pass `value` as a fraction (0.125 for 12.5%) when
 * `asFraction` is true, or pre-multiplied when false (12.5 for 12.5%).
 */
export function formatPercentage(
  value: number | null | undefined,
  locale: Locale,
  fractionDigits = 2,
  asFraction = false
): string {
  if (value === null || value === undefined || !isFinite(value)) return '-';
  const pct = asFraction ? value * 100 : value;
  return (
    formatNumber(pct, locale, {
      minimumFractionDigits: fractionDigits,
      maximumFractionDigits: fractionDigits,
    }) + '%'
  );
}

/**
 * Format age range. Callers should pass the translated "years" label
 * via the yearsText parameter (e.g. t('common.years')).
 */
export function formatAgeRange(
  minAge: number | null | undefined,
  maxAge: number | null | undefined,
  yearsText = 'years'
): string {
  if (minAge === null && maxAge === null) return '-';
  if (minAge === null || minAge === undefined) return `< ${maxAge} ${yearsText}`;
  if (maxAge === null || maxAge === undefined) return `${minAge}+ ${yearsText}`;
  return `${minAge}-${maxAge} ${yearsText}`;
}

/**
 * Format an age range in months (e.g., "12–36", "12+", "0–24").
 * Returns null if both min and max are absent.
 */
export function formatMonthRange(min?: number | null, max?: number | null): string | null {
  if (min == null && max == null) return null;
  if (min != null && max != null) return `${min}\u2013${max}`;
  if (min != null) return `${min}+`;
  return `0\u2013${max}`;
}

/**
 * Returns the first day of the current month as a YYYY-MM-DD string.
 *
 * "Current" in Europe/Berlin, like every other date decision here -- these feed
 * the statistics endpoints' `from`/`to`, and the server snaps its own ranges
 * with `models.Today()`.
 */
export function getCurrentMonthStart(): string {
  return `${todayBerlinString().slice(0, 7)}-01`;
}

/**
 * Returns the first and last day of the current month as YYYY-MM-DD strings.
 */
export function getCurrentMonthRange(): { from: string; to: string } {
  const [year, month] = todayBerlinString().split('-').map(Number);
  const pad = (n: number) => n.toString().padStart(2, '0');
  // Day 0 of the next month is the last day of this one, and Date.UTC
  // normalises December into the following January for us.
  const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return {
    from: `${year}-${pad(month)}-01`,
    to: `${year}-${pad(month)}-${pad(lastDay)}`,
  };
}

/**
 * Format an ISO datetime string to HH:mm in local time.
 * Parses the ISO string into a Date object so timezone conversion is handled correctly.
 */
export function formatTime(isoString: string | null | undefined): string {
  if (!isoString) return '';
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return '';
  const hh = date.getHours().toString().padStart(2, '0');
  const mm = date.getMinutes().toString().padStart(2, '0');
  return `${hh}:${mm}`;
}

/**
 * Combine a date string (YYYY-MM-DD) and time string (HH:mm) into an ISO datetime string.
 * Parses both as local time and converts to UTC for proper timezone handling.
 * Returns null if time is empty.
 */
export function combineDateAndTime(dateStr: string, timeStr: string): string | null {
  if (!timeStr) return null;
  const date = new Date(`${dateStr}T${timeStr}:00`);
  if (isNaN(date.getTime())) return null;
  return date.toISOString();
}

/**
 * Format a Date object to YYYY-MM-DD using local timezone getters.
 * This is the timezone-safe replacement for `date.toISOString().slice(0, 10)`.
 */
export function toLocalDateString(date: Date): string {
  const y = date.getFullYear();
  const m = (date.getMonth() + 1).toString().padStart(2, '0');
  const d = date.getDate().toString().padStart(2, '0');
  return `${y}-${m}-${d}`;
}

// Re-export contract properties utilities for backwards compatibility
export {
  propertiesToValues,
  getPropertyValue,
  getScalarPropertyValue,
  setProperty,
  removePropertyByValue,
  hasPropertyValue,
  getKeyForValue,
} from './contract-properties';
export type { ContractProperties, FundingAttribute } from './contract-properties';
