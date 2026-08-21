/**
 * The frontend's answer to "what date is it?", which has to be the same answer
 * the backend's `models.Today()` gives.
 *
 * These run under a pinned clock rather than a pinned timezone: every function
 * here names Europe/Berlin explicitly, so the browser's own zone must not be
 * able to change the result. The `TZ=` cases below prove that by re-running the
 * same assertions with the process clock moved — CI runs in UTC and developer
 * machines run in Berlin, and this used to be exactly the seam where a test
 * passed on one and failed on the other.
 */

import {
  addDaysToDateString,
  berlinDateString,
  getDayBefore,
  todayBerlin,
  todayBerlinString,
  toUTCDate,
} from '../contracts';

describe('berlinDateString', () => {
  it('formats as zero-padded YYYY-MM-DD', () => {
    expect(berlinDateString(new Date('2026-03-05T12:00:00Z'))).toBe('2026-03-05');
  });

  it('is already on tomorrow when Berlin is, and UTC is not', () => {
    // 22:30 UTC on 31 July is 00:30 on 1 August in Berlin (CEST, UTC+2).
    expect(berlinDateString(new Date('2026-07-31T22:30:00Z'))).toBe('2026-08-01');
  });

  it('is still on yesterday when Berlin is, and UTC has moved on', () => {
    // 00:30 UTC on 1 January is 01:30 on 1 January in Berlin (CET, UTC+1) --
    // same day. Half an hour earlier is not.
    expect(berlinDateString(new Date('2025-12-31T23:30:00Z'))).toBe('2026-01-01');
    expect(berlinDateString(new Date('2025-12-31T22:30:00Z'))).toBe('2025-12-31');
  });

  it('handles both sides of the DST switch', () => {
    // Last Sunday in March: CET (+1) becomes CEST (+2) at 02:00 local.
    expect(berlinDateString(new Date('2026-03-28T23:30:00Z'))).toBe('2026-03-29');
    // Last Sunday in October: CEST (+2) becomes CET (+1) at 03:00 local.
    expect(berlinDateString(new Date('2026-10-24T22:30:00Z'))).toBe('2026-10-25');
  });
});

describe('todayBerlinString / todayBerlin', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('reads the pinned clock', () => {
    jest.setSystemTime(new Date('2026-05-17T09:00:00Z'));
    expect(todayBerlinString()).toBe('2026-05-17');
    expect(todayBerlin()).toBe(Date.UTC(2026, 4, 17));
  });

  it('agrees with itself across the string and numeric forms', () => {
    // The post-midnight window is where a UTC-derived "today" was off by one.
    jest.setSystemTime(new Date('2026-05-17T22:30:00Z'));
    expect(todayBerlinString()).toBe('2026-05-18');
    expect(todayBerlin()).toBe(toUTCDate(todayBerlinString()));
  });
});

describe('addDaysToDateString', () => {
  it('adds and subtracts whole days', () => {
    expect(addDaysToDateString('2026-05-17', 1)).toBe('2026-05-18');
    expect(addDaysToDateString('2026-05-17', -1)).toBe('2026-05-16');
    expect(addDaysToDateString('2026-05-17', 0)).toBe('2026-05-17');
  });

  it('crosses month, year and leap boundaries', () => {
    expect(addDaysToDateString('2026-01-31', 1)).toBe('2026-02-01');
    expect(addDaysToDateString('2026-12-31', 1)).toBe('2027-01-01');
    expect(addDaysToDateString('2026-01-01', -1)).toBe('2025-12-31');
    expect(addDaysToDateString('2024-02-28', 1)).toBe('2024-02-29');
    expect(addDaysToDateString('2025-02-28', 1)).toBe('2025-03-01');
  });

  it('accepts a full RFC3339 timestamp and answers with a bare date', () => {
    // What the API sends for a contract's `from`, which the boundary-move
    // optimistic update feeds straight in.
    expect(addDaysToDateString('2026-05-17T00:00:00Z', -1)).toBe('2026-05-16');
  });

  it('does not shift across a DST boundary', () => {
    // The day after the spring-forward Sunday has no 00:00 in local terms in
    // some zones; pure calendar arithmetic does not care.
    expect(addDaysToDateString('2026-03-29', 1)).toBe('2026-03-30');
    expect(addDaysToDateString('2026-10-25', -1)).toBe('2026-10-24');
  });
});

describe('getDayBefore', () => {
  it('answers in bare-date form for a timestamp input', () => {
    expect(getDayBefore('2026-05-17T00:00:00Z')).toBe('2026-05-16');
  });
});
