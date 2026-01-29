/**
 * Format a number in cents to a currency string (EUR)
 */
export function formatCurrency(cents: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR'
  }).format(cents / 100)
}

/**
 * Format a date string to a localized date string
 * @param dateStr - The date string to format
 * @param locale - The locale for formatting (default: 'de-DE')
 * @param fallback - The string to return when dateStr is null/undefined (default: '-')
 */
export function formatDate(
  dateStr: string | null | undefined,
  locale = 'de-DE',
  fallback = '-'
): string {
  if (!dateStr) return fallback
  return new Date(dateStr).toLocaleDateString(locale)
}

/**
 * Format a Date object to ISO date string (YYYY-MM-DD)
 */
export function formatDateToISO(date: Date): string {
  return date.toISOString().split('T')[0]
}

/**
 * Calculate age from a birthdate string
 */
export function calculateAge(birthdate: string): number {
  const birth = new Date(birthdate)
  const today = new Date()
  let age = today.getFullYear() - birth.getFullYear()
  const monthDiff = today.getMonth() - birth.getMonth()
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birth.getDate())) {
    age--
  }
  return age
}
