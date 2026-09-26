/**
 * Parse a date string to UTC start-of-day timestamp (milliseconds).
 * Handles both "2025-01-01" and "2025-01-01T00:00:00Z" formats.
 */
export function toUTCDate(d: string): number {
  const date = new Date(d);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate());
}

/**
 * The calendar date an instant falls on in Europe/Berlin, as "YYYY-MM-DD".
 *
 * This is the frontend's `models.Today()`: the one place that answers "what
 * date is it?", anchored to the application timezone rather than to whatever
 * zone the browser happens to be in. Everything that needs today — list
 * filters, contract-start defaults, "fetch the active roster" — goes through
 * this or `todayBerlin()` below, so that the client and the server never
 * disagree about which day it is.
 *
 * Two ways they used to disagree, both real:
 *
 *   - The browser is behind Berlin. At 20:00 in New York it is already 02:00
 *     the next day in Berlin, so a "tomorrow" default computed from the local
 *     clock lands on the day the server already calls today — and the amend
 *     threshold, which compares against `models.Today()`, rejects it.
 *   - The browser *is* Berlin but the code derived the day from UTC. Between
 *     midnight and 01:00/02:00 local, UTC is still on yesterday.
 *
 * `en-CA` is not a display choice: its short date format is ISO 8601, so
 * `formatToParts` hands back zero-padded numeric year/month/day with no
 * locale-specific reordering to undo.
 */
export function berlinDateString(instant: Date = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Europe/Berlin',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(instant);
  const get = (type: string) => parts.find((p) => p.type === type)!.value;
  return `${get('year')}-${get('month')}-${get('day')}`;
}

/** Today's calendar date in Europe/Berlin, as "YYYY-MM-DD". */
export function todayBerlinString(): string {
  return berlinDateString();
}

/**
 * Today's calendar date in Europe/Berlin, as a `Date` at local midnight.
 *
 * For the places that hold a date in component state and hand it to a picker or
 * a `date-fns` calculation. Local midnight rather than UTC midnight because
 * that is what `toLocalDateString` and `date-fns` read back, so the value
 * survives the round trip as the same calendar day.
 */
export function todayBerlinDate(): Date {
  return new Date(`${todayBerlinString()}T00:00:00`);
}

/**
 * Start-of-day timestamp for "today" as a calendar date in Europe/Berlin.
 *
 * The numeric counterpart of `todayBerlinString()`, returned as a UTC-midnight
 * timestamp so it compares directly against `toUTCDate(period.from)`.
 */
export function todayBerlin(): number {
  return toUTCDate(todayBerlinString());
}

/**
 * Shift a "YYYY-MM-DD" date string by whole days, returning the same shape.
 *
 * Pure calendar arithmetic on the date parts — no local-midnight `Date` in the
 * middle, which is what made the older helpers shift by an extra day in zones
 * behind UTC. Handles month, year and leap boundaries via `Date.UTC`'s own
 * normalisation.
 */
export function addDaysToDateString(dateStr: string, days: number): string {
  const [y, m, d] = dateStr.slice(0, 10).split('-').map(Number);
  const shifted = new Date(Date.UTC(y, m - 1, d + days));
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}`;
}

/**
 * Resolve an `asOf` argument to the UTC-midnight timestamp to compare against.
 *
 * Omitted means today in Europe/Berlin, which is what every caller without a
 * date control wants. A caller *with* one passes the date it is displaying, so
 * that "which contract applies" is answered for the row the user is looking at
 * rather than for the day they happen to be looking on.
 *
 * An unparseable value falls back to today rather than propagating NaN. Every
 * comparison against NaN is false, so the alternative is a screen that silently
 * shows nothing active — a blank kanban board, a roster with no sections — with
 * no error to explain it. A date input cleared to "" is enough to reach here.
 */
function asOfTimestamp(asOf?: string | null): number {
  if (!asOf) return todayBerlin();
  const parsed = toUTCDate(asOf);
  return Number.isNaN(parsed) ? todayBerlin() : parsed;
}

/**
 * Is a period (from/to) active on `asOf` — by default, today in Europe/Berlin?
 *
 * Both ends are inclusive days; a missing `to` means the period never ends.
 */
export function isActivePeriod(
  period: { from: string; to?: string | null },
  asOf?: string | null
): boolean {
  const on = asOfTimestamp(asOf);
  return toUTCDate(period.from) <= on && (!period.to || toUTCDate(period.to) >= on);
}

/**
 * The contract active on `asOf` — by default, today in Europe/Berlin.
 *
 * Pass `asOf` from any view with a date control. The list endpoints filter the
 * *rows* by `active_on` but hand back each entity's full contract history
 * (`Preload("Contracts")` with no filter), so resolving against today would
 * describe a historical row with the contract that applies now: the section the
 * child has since moved to, the salary the employee is on today. Where a date is
 * in play it has to reach here too, or the page contradicts its own filter.
 */
export function getActiveContract<T extends { from: string; to?: string | null }>(
  contracts?: T[],
  asOf?: string | null
): T | null {
  if (!contracts || contracts.length === 0) return null;
  return contracts.find((c) => isActivePeriod(c, asOf)) || null;
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
 * The contract in force on `asOf`, or the nearest thing to it.
 *
 * Like `getActiveContract`, but never answers null for an entity that has any
 * contracts at all — the list tables need *something* to render in the section
 * and grade columns. `asOf` defaults to today in Europe/Berlin.
 *
 * The fallback is the latest contract that had already started by `asOf`, so a
 * gap between two contracts is described by the one that just ended rather than
 * by one that has not begun. Only when every contract starts after `asOf` does
 * it answer with the earliest, which is then the next one due.
 *
 * On the list pages the fallback is unreachable by construction: the server
 * filtered those rows by `active_on`, so a contract covering `asOf` exists. It
 * earns its keep for callers holding an entity fetched some other way.
 */
export function getCurrentContract<T extends { from: string; to?: string | null }>(
  contracts?: T[],
  asOf?: string | null
): T | null {
  if (!contracts || contracts.length === 0) return null;
  const active = contracts.find((c) => isActivePeriod(c, asOf));
  if (active) return active;

  const on = asOfTimestamp(asOf);
  const byStartDesc = [...contracts].sort((a, b) => toUTCDate(b.from) - toUTCDate(a.from));
  return byStartDesc.find((c) => toUTCDate(c.from) <= on) ?? byStartDesc[byStartDesc.length - 1];
}

/**
 * Get the day before a given date string.
 *
 * Accepts either "YYYY-MM-DD" or a full RFC3339 timestamp, and answers in
 * "YYYY-MM-DD" — callers pass both, since a contract's `from` arrives from the
 * API as `2025-01-15T00:00:00Z` but is edited as a bare date.
 */
export function getDayBefore(dateStr: string): string {
  return addDaysToDateString(dateStr, -1);
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

/**
 * Do two contract periods cover any calendar day in common?
 *
 * Both ends are inclusive days, and a missing `to` means the period never ends
 * — so an open-ended contract overlaps everything starting on or after its
 * `from`. That is the same rule the server's exclusion constraint applies, which
 * is the point: the dialogs use this to stop a submission the database is going
 * to refuse anyway, instead of spending a round trip to find out.
 *
 * A period whose `from` does not parse is not claimed to overlap anything. An
 * empty start date is the required-field validation's business, and answering
 * "yes, overlaps" for it would block the form on a message about the wrong
 * problem.
 */
export function periodsOverlap(
  a: { from: string; to?: string | null },
  b: { from: string; to?: string | null }
): boolean {
  const aFrom = toUTCDate(a.from);
  const bFrom = toUTCDate(b.from);
  if (Number.isNaN(aFrom) || Number.isNaN(bFrom)) return false;
  const aTo = a.to ? toUTCDate(a.to) : Infinity;
  const bTo = b.to ? toUTCDate(b.to) : Infinity;
  return aFrom <= bTo && bFrom <= aTo;
}
