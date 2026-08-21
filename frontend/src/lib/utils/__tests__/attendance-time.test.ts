import { checkOutTimestampOnDate, timestampOnDate } from '../attendance-time';
import { formatTime } from '../formatting';

/** The local calendar date of an instant, which is what `formatTime` reads against. */
function localDate(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

describe('timestampOnDate', () => {
  it('keeps the time of day and moves the date', () => {
    // Someone standing in the group room on Friday afternoon, filling in
    // Monday's column.
    const friday = new Date('2026-05-15T14:32:10');
    const stamped = timestampOnDate('2026-05-11', friday);

    expect(localDate(stamped)).toBe('2026-05-11');
    expect(formatTime(stamped)).toBe(formatTime(friday.toISOString()));
  });

  it('is a no-op on the date it is already on', () => {
    // Checking a child in today must behave exactly as it always did.
    const now = new Date('2026-05-15T08:03:00');
    expect(timestampOnDate('2026-05-15', now)).toBe(now.toISOString());
  });

  it('moves forward as readily as back', () => {
    const monday = new Date('2026-05-11T09:00:00');
    expect(localDate(timestampOnDate('2026-05-15', monday))).toBe('2026-05-15');
  });

  it('crosses month and year boundaries', () => {
    const instant = new Date('2026-01-02T07:15:00');
    expect(localDate(timestampOnDate('2025-12-31', instant))).toBe('2025-12-31');
    expect(localDate(timestampOnDate('2026-02-28', instant))).toBe('2026-02-28');
  });

  it('lands on 29 February in a leap year', () => {
    const instant = new Date('2024-03-04T10:00:00');
    expect(localDate(timestampOnDate('2024-02-29', instant))).toBe('2024-02-29');
  });

  it('accepts a full RFC3339 date, not just a bare one', () => {
    const instant = new Date('2026-05-15T14:00:00');
    expect(localDate(timestampOnDate('2026-05-11T00:00:00Z', instant))).toBe('2026-05-11');
  });

  it('produces a value that reads back as the same clock time', () => {
    // The round-trip that matters: the grid writes this and renders it with
    // formatTime. If the two disagreed, an edit would silently move the time.
    const instant = new Date('2026-05-15T16:45:00');
    const stamped = timestampOnDate('2026-05-11', instant);
    expect(formatTime(stamped)).toBe('16:45');
  });
});

describe('checkOutTimestampOnDate', () => {
  it('is the current time of day on the recorded date', () => {
    const now = new Date('2026-05-15T16:20:00');
    const checkIn = timestampOnDate('2026-05-11', new Date('2026-05-15T08:00:00'));

    const checkOut = checkOutTimestampOnDate('2026-05-11', checkIn, now);

    expect(localDate(checkOut)).toBe('2026-05-11');
    expect(formatTime(checkOut)).toBe('16:20');
    expect(new Date(checkOut).getTime()).toBeGreaterThan(new Date(checkIn).getTime());
  });

  it('clamps when local midnight has passed since check-in', () => {
    // Checked a past day in at 23:59, clicking check-out at 00:00. Both land on
    // the recorded date, so the naive answer sorts before the check-in and the
    // server rejects the update.
    const checkIn = timestampOnDate('2026-05-11', new Date('2026-05-15T23:59:30'));
    const justAfterMidnight = new Date('2026-05-16T00:00:10');

    const checkOut = checkOutTimestampOnDate('2026-05-11', checkIn, justAfterMidnight);

    expect(new Date(checkOut).getTime()).toBeGreaterThan(new Date(checkIn).getTime());
    expect(new Date(checkOut).getTime()).toBe(new Date(checkIn).getTime() + 60_000);
  });

  it('clamps when check-out lands on exactly the check-in instant', () => {
    // The server requires strictly before, not before-or-equal.
    const checkIn = timestampOnDate('2026-05-11', new Date('2026-05-15T10:00:00'));
    const sameMoment = new Date('2026-05-15T10:00:00');

    const checkOut = checkOutTimestampOnDate('2026-05-11', checkIn, sameMoment);

    expect(new Date(checkOut).getTime()).toBeGreaterThan(new Date(checkIn).getTime());
  });

  it('does not clamp when there is no check-in to sort against', () => {
    const now = new Date('2026-05-15T09:00:00');
    expect(checkOutTimestampOnDate('2026-05-11', null, now)).toBe(
      timestampOnDate('2026-05-11', now)
    );
    expect(checkOutTimestampOnDate('2026-05-11', undefined, now)).toBe(
      timestampOnDate('2026-05-11', now)
    );
  });

  it('ignores an unparseable check-in rather than throwing', () => {
    const now = new Date('2026-05-15T09:00:00');
    expect(checkOutTimestampOnDate('2026-05-11', 'not-a-date', now)).toBe(
      timestampOnDate('2026-05-11', now)
    );
  });
});
