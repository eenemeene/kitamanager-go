/**
 * Parse a date string to UTC start-of-day timestamp (milliseconds).
 * Handles both "2025-01-01" and "2025-01-01T00:00:00Z" formats.
 */
export function toUTCDate(d: string): number {
  const date = new Date(d);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate());
}

/**
 * Get UTC start-of-day timestamp for today.
 */
function todayUTC(): number {
  const now = new Date();
  return Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
}

/**
 * Check if a period (from/to) is active today.
 */
export function isActivePeriod(period: { from: string; to?: string | null }): boolean {
  const today = todayUTC();
  return toUTCDate(period.from) <= today && (!period.to || toUTCDate(period.to) >= today);
}

/**
 * Get the currently active contract (from <= today, no end date or end date >= today)
 */
export function getActiveContract<T extends { from: string; to?: string | null }>(
  contracts?: T[]
): T | null {
  if (!contracts || contracts.length === 0) return null;
  return contracts.find((c) => isActivePeriod(c)) || null;
}

/**
 * Get the current or most recent contract.
 * Falls back to the contract with the latest start date.
 */
export function getCurrentContract<T extends { from: string; to?: string | null }>(
  contracts?: T[]
): T | null {
  if (!contracts || contracts.length === 0) return null;
  return (
    contracts.find((c) => isActivePeriod(c)) ||
    [...contracts].sort((a, b) => toUTCDate(b.from) - toUTCDate(a.from))[0]
  );
}

/**
 * Get the day before a given date string (YYYY-MM-DD format)
 */
export function getDayBefore(dateStr: string): string {
  const date = new Date(dateStr);
  date.setDate(date.getDate() - 1);
  return date.toISOString().split('T')[0];
}

/**
 * Get the status of a contract relative to today
 */
export function getContractStatus(
  contract: { from: string; to?: string | null } | null
): 'active' | 'upcoming' | 'ended' | null {
  if (!contract) return null;
  const today = todayUTC();
  if (toUTCDate(contract.from) > today) return 'upcoming';
  if (contract.to && toUTCDate(contract.to) < today) return 'ended';
  return 'active';
}

/**
 * Compare two date strings for sorting (ascending).
 * Returns negative if a < b, positive if a > b, 0 if equal.
 */
export function compareDates(a: string, b: string): number {
  return toUTCDate(a) - toUTCDate(b);
}

/**
 * Check if date a is before date b.
 */
export function isDateBefore(a: string, b: string): boolean {
  return toUTCDate(a) < toUTCDate(b);
}
