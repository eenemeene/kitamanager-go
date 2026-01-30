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
