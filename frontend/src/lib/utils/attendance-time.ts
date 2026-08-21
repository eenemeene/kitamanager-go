/**
 * Timestamps for attendance records.
 *
 * An attendance row carries a calendar `date` and, when the child was present,
 * a `check_in_time` / `check_out_time` instant. Those two have to agree, and
 * nothing enforces it: the API takes the timestamp as given
 * (internal/service/attendance.go), and the UI renders only `HH:mm`
 * (`formatTime`), so a timestamp on the wrong day looks perfectly ordinary on
 * screen.
 *
 * The week grid offers a check-in button in every Mon–Fri cell and the stepper
 * reaches any week, so "the day being recorded" is routinely not today. Stamping
 * `new Date()` therefore wrote Friday's instant onto Monday's row. It showed as
 * a plausible time, and it broke on the next edit: correcting the check-out to
 * "16:00" builds an instant on *Monday*, which is before the stored Friday
 * check-in, and the server rejects the whole update with "check-in time must be
 * before check-out time".
 */

/**
 * The current wall-clock time of day, moved onto `dateStr`.
 *
 * Local-zone arithmetic on purpose: the value is read back through
 * `formatTime`, which renders local `HH:mm`. Building it any other way would
 * make the round-trip disagree with itself.
 *
 * @param dateStr the record's calendar date, "YYYY-MM-DD" (or an RFC3339 prefix)
 * @param instant the moment to take the time of day from; defaults to now
 * @returns an RFC3339 timestamp
 */
export function timestampOnDate(dateStr: string, instant: Date = new Date()): string {
  const [year, month, day] = dateStr.slice(0, 10).split('-').map(Number);
  const stamped = new Date(instant);
  // Sets the local date while leaving the local time of day alone. Across a DST
  // boundary the wall clock is preserved and the offset moves, which is what
  // "the same time of day, on that date" means.
  stamped.setFullYear(year, month - 1, day);
  return stamped.toISOString();
}

/** One minute, in milliseconds — the nudge applied when a check-out would not sort after its check-in. */
const MINIMUM_STAY_MS = 60_000;

/**
 * A check-out timestamp for `dateStr` that is guaranteed to sort after
 * `checkInIso`.
 *
 * Normally just the current time of day on that date. The clamp covers one
 * genuine edge: recording a past day across local midnight. A child checked in
 * at 23:59 and checked out at 00:00 both land on the recorded date, in that
 * order reversed — the server would reject the update, and the user would have
 * no idea why. A minute after check-in is the smallest honest answer, and the
 * time is editable afterwards either way.
 */
export function checkOutTimestampOnDate(
  dateStr: string,
  checkInIso: string | null | undefined,
  instant: Date = new Date()
): string {
  const checkOut = timestampOnDate(dateStr, instant);
  if (!checkInIso) return checkOut;

  const checkInMs = new Date(checkInIso).getTime();
  if (Number.isNaN(checkInMs)) return checkOut;

  if (new Date(checkOut).getTime() > checkInMs) return checkOut;
  return new Date(checkInMs + MINIMUM_STAY_MS).toISOString();
}
