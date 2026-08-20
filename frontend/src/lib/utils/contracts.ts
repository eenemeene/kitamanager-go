/**
 * Parse a date string to UTC start-of-day timestamp (milliseconds).
 * Handles both "2025-01-01" and "2025-01-01T00:00:00Z" formats.
 */
export function toUTCDate(d: string): number {
  const date = new Date(d);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate());
}

/**
 * Start-of-day timestamp for "today" as a calendar date in Europe/Berlin.
 *
 * This mirrors the backend's `models.Today()` (the canonical "what date is
 * it?" source, defaulting to Europe/Berlin). Deriving "today" from the
 * browser's *UTC* day instead would be off by one for a Berlin user in the
 * post-midnight window (00:00–01:00/02:00 local is still "yesterday" in UTC),
 * making contract-status badges and the active-contract pick disagree with
 * the server. Returned as a UTC-midnight timestamp so it compares directly
 * against `toUTCDate(period.from)`.
 */
export function todayBerlin(): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'Europe/Berlin',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date());
  const get = (type: string) => Number(parts.find((p) => p.type === type)!.value);
  return Date.UTC(get('year'), get('month') - 1, get('day'));
}

/**
 * Check if a period (from/to) is active today.
 */
export function isActivePeriod(period: { from: string; to?: string | null }): boolean {
  const today = todayBerlin();
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
 * Why a child's contract wants attention against their expected school start.
 *
 * Two distinct situations, deliberately not collapsed into one boolean: the
 * sentence you can truthfully say about each is different. A contract that ends
 * too late *ends* — you can name the date and say it runs past school start. An
 * open-ended one does not end at all, so the same sentence would be false, which
 * is why it was previously excluded from the warning altogether rather than
 * given wording of its own.
 *
 * Excluding it is the worse failure. Open-ended is a first-class state here
 * (nullable `to_date`, optional in the create form, and `ContractEndRequest`
 * treats a null `to` as "reopen indefinitely"), and a child who left for school
 * on one keeps being counted until somebody notices by hand. That is precisely
 * what this warning exists to prevent, so it should fire hardest there.
 *
 * Anchored to Europe/Berlin, like every other "is it today yet" decision here.
 */
export type SchoolOverrun = 'ends-after-school-start' | 'no-end-past-school-start' | null;

export function classifySchoolOverrun(
  contract: { to?: string | null } | null | undefined,
  mussContractEnd: string
): SchoolOverrun {
  if (!contract) return null;
  const schoolEnd = toUTCDate(mussContractEnd);
  if (contract.to) {
    return toUTCDate(contract.to) > schoolEnd ? 'ends-after-school-start' : null;
  }
  // No end date: only worth flagging once the expected school start has
  // actually passed. Before then an open-ended contract is ordinary.
  return todayBerlin() > schoolEnd ? 'no-end-past-school-start' : null;
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
  const date = new Date(`${dateStr}T00:00:00`);
  date.setDate(date.getDate() - 1);
  const y = date.getFullYear();
  const m = (date.getMonth() + 1).toString().padStart(2, '0');
  const d = date.getDate().toString().padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/**
 * Get the status of a contract relative to today
 */
export function getContractStatus(
  contract: { from: string; to?: string | null } | null
): 'active' | 'upcoming' | 'ended' | null {
  if (!contract) return null;
  const today = todayBerlin();
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
