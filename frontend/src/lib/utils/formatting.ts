import { format, parseISO, differenceInYears, type Locale } from 'date-fns';
import { de, enUS } from 'date-fns/locale';

const locales: Record<string, Locale> = {
  de: de,
  en: enUS,
};

/**
 * Format a date string for display
 */
export function formatDate(dateString: string | null | undefined, locale = 'en'): string {
  if (!dateString) return '-';
  try {
    const date = parseISO(dateString);
    return format(date, 'PP', { locale: locales[locale] || enUS });
  } catch {
    return dateString;
  }
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
export function formatCurrency(cents: number | null | undefined, locale = 'de'): string {
  if (cents === null || cents === undefined) return '-';
  const euros = cents / 100;
  return new Intl.NumberFormat(locale === 'de' ? 'de-DE' : 'en-US', {
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
  locale = 'en',
  ongoingText = 'ongoing'
): string {
  const fromFormatted = formatDate(from, locale);
  const toFormatted = to ? formatDate(to, locale) : ongoingText;
  return `${fromFormatted} - ${toFormatted}`;
}

/**
 * Format FTE (Full Time Equivalent) / staffing ratio
 */
export function formatFte(ratio: number): string {
  return ratio.toFixed(2);
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
 */
export function getCurrentMonthStart(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`;
}

/**
 * Returns the first and last day of the current month as YYYY-MM-DD strings.
 */
export function getCurrentMonthRange(): { from: string; to: string } {
  const now = new Date();
  const y = now.getFullYear();
  const m = now.getMonth();
  const first = new Date(y, m, 1);
  const last = new Date(y, m + 1, 0);
  const pad = (n: number) => n.toString().padStart(2, '0');
  return {
    from: `${first.getFullYear()}-${pad(first.getMonth() + 1)}-${pad(first.getDate())}`,
    to: `${last.getFullYear()}-${pad(last.getMonth() + 1)}-${pad(last.getDate())}`,
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
