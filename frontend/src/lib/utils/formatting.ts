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
 * Format age range
 */
export function formatAgeRange(
  minAge: number | null | undefined,
  maxAge: number | null | undefined,
  locale = 'en'
): string {
  const yearsText = locale === 'de' ? 'Jahre' : 'years';

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
  return {
    from: first.toISOString().slice(0, 10),
    to: last.toISOString().slice(0, 10),
  };
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
