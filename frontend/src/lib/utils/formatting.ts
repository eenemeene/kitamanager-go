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
 * Contract properties utilities
 * Properties are stored as {key: value} where key is the category
 * and value is the selected option. Keys are mutually exclusive
 * (e.g., care_type can only have one value).
 * Values can be strings (scalar) or string arrays.
 */

export type ContractProperties = Record<string, string | string[]>;

export interface FundingAttribute {
  key: string;
  value: string;
}

/**
 * Convert contract properties map to a flat array of values.
 * Used for displaying selected attributes in the UI.
 * Handles both scalar strings and string arrays.
 */
export function propertiesToValues(properties?: ContractProperties): string[] {
  if (!properties) return [];
  const result: string[] = [];
  for (const value of Object.values(properties)) {
    if (Array.isArray(value)) {
      result.push(...value);
    } else {
      result.push(value);
    }
  }
  return result;
}

/**
 * Get the value for a specific key from properties.
 * Returns the value if it's a string, or undefined if it's an array or doesn't exist.
 */
export function getPropertyValue(
  properties: ContractProperties | undefined,
  key: string
): string | string[] | undefined {
  return properties?.[key];
}

/**
 * Get scalar value for a specific key from properties.
 * Returns the value only if it's a string, not an array.
 */
export function getScalarPropertyValue(
  properties: ContractProperties | undefined,
  key: string
): string | undefined {
  const value = properties?.[key];
  return typeof value === 'string' ? value : undefined;
}

/**
 * Set a property value. If the key already exists, it replaces the value.
 * Returns a new properties object.
 */
export function setProperty(
  properties: ContractProperties | undefined,
  key: string,
  value: string | string[]
): ContractProperties {
  return { ...properties, [key]: value };
}

/**
 * Remove a property by its value. Finds the key that has this value and removes it.
 * Returns a new properties object.
 */
export function removePropertyByValue(
  properties: ContractProperties | undefined,
  value: string
): ContractProperties | undefined {
  if (!properties) return undefined;

  const newProps: ContractProperties = {};
  for (const [k, v] of Object.entries(properties)) {
    if (v !== value) {
      newProps[k] = v;
    }
  }
  return Object.keys(newProps).length > 0 ? newProps : undefined;
}

/**
 * Check if properties contain a specific value.
 */
export function hasPropertyValue(
  properties: ContractProperties | undefined,
  value: string
): boolean {
  if (!properties) return false;
  return Object.values(properties).includes(value);
}

/**
 * Get the key for a value in properties.
 */
export function getKeyForValue(
  properties: ContractProperties | undefined,
  value: string
): string | undefined {
  if (!properties) return undefined;
  for (const [k, v] of Object.entries(properties)) {
    if (v === value) return k;
  }
  return undefined;
}
