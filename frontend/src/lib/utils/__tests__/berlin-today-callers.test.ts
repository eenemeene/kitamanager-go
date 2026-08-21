/**
 * The callers that have to agree with the server about what day it is.
 *
 * `models.Today()` is the backend's only answer to that question and it is
 * anchored to Europe/Berlin. Anything the frontend sends as a date — a list
 * filter, a contract start, a statistics range — is compared against it, so a
 * value derived from the browser's clock is a disagreement waiting for a user
 * in the wrong timezone or the wrong hour.
 *
 * Every assertion here pins the clock to a moment where Berlin and UTC are on
 * *different* calendar days, which is the only interesting case: at 22:30 UTC
 * in summer, Berlin is already on tomorrow.
 */

import { getCurrentMonthRange, getCurrentMonthStart } from '../formatting';
import { calculateYearsOfService } from '../step-promotions';
import { todayBerlinDate, todayBerlinString } from '../contracts';

/** 00:30 on 1 August in Berlin (CEST); still 31 July in UTC. */
const BERLIN_IS_AHEAD = new Date('2026-07-31T22:30:00Z');

describe('date defaults are anchored to Berlin', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('the month helpers roll over when Berlin does', () => {
    // These feed the statistics endpoints' from/to, and the server snaps its
    // own range with models.Today(). A month behind is a whole page of wrong
    // numbers, not an off-by-one day.
    jest.setSystemTime(BERLIN_IS_AHEAD);

    expect(todayBerlinString()).toBe('2026-08-01');
    expect(getCurrentMonthStart()).toBe('2026-08-01');
    expect(getCurrentMonthRange()).toEqual({ from: '2026-08-01', to: '2026-08-31' });
  });

  it('the month range ends on the real last day, including February', () => {
    jest.setSystemTime(new Date('2024-02-10T12:00:00Z'));
    expect(getCurrentMonthRange()).toEqual({ from: '2024-02-01', to: '2024-02-29' });

    jest.setSystemTime(new Date('2025-02-10T12:00:00Z'));
    expect(getCurrentMonthRange()).toEqual({ from: '2025-02-01', to: '2025-02-28' });
  });

  it('the month range does not run past December', () => {
    jest.setSystemTime(new Date('2026-12-10T12:00:00Z'));
    expect(getCurrentMonthRange()).toEqual({ from: '2026-12-01', to: '2026-12-31' });
  });

  it('todayBerlinDate reads back as the same calendar day', () => {
    // It is handed to date pickers and date-fns, which read local components.
    jest.setSystemTime(BERLIN_IS_AHEAD);
    const today = todayBerlinDate();

    expect(today.getFullYear()).toBe(2026);
    expect(today.getMonth()).toBe(7); // August
    expect(today.getDate()).toBe(1);
  });

  it('years of service are measured from Berlin today by default', () => {
    // The eligible step this feeds is compared against the server's answer; a
    // day either side of an anniversary is a disagreement about a pay grade.
    jest.setSystemTime(BERLIN_IS_AHEAD);
    const contracts = [{ from: '2021-08-01' }];

    const byDefault = calculateYearsOfService(contracts);

    // It measures to Berlin's day...
    expect(byDefault).toBeCloseTo(
      calculateYearsOfService(contracts, new Date('2026-08-01T00:00:00Z')),
      10
    );
    // ...and not to the UTC one, which is still on the previous day here.
    expect(byDefault).not.toBeCloseTo(
      calculateYearsOfService(contracts, new Date('2026-07-31T00:00:00Z')),
      6
    );
  });

  it('years of service is still overridable for a specific date', () => {
    jest.setSystemTime(BERLIN_IS_AHEAD);
    const years = calculateYearsOfService(
      [{ from: '2020-01-01' }],
      new Date('2022-01-01T00:00:00Z')
    );
    expect(Math.round(years)).toBe(2);
  });

  it('a contract starting today counts as no service, in any browser timezone', () => {
    // The two dates have to be compared in one frame: contract dates parse as
    // UTC midnight, so "today" has to be UTC midnight of Berlin's day, not
    // local midnight of it. Mixing the two credited a few hours of service to a
    // contract that had not started.
    jest.setSystemTime(new Date('2026-06-15T09:00:00Z'));
    expect(calculateYearsOfService([{ from: '2026-06-15' }])).toBe(0);
  });
});
