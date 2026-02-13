import {
  getActiveContract,
  getCurrentContract,
  getDayBefore,
  getContractStatus,
  isActivePeriod,
  compareDates,
  isDateBefore,
  toUTCDate,
} from '../contracts';

// ---------------------------------------------------------------------------
// toUTCDate
// ---------------------------------------------------------------------------
describe('toUTCDate', () => {
  it('parses YYYY-MM-DD to UTC midnight', () => {
    expect(toUTCDate('2025-06-15')).toBe(Date.UTC(2025, 5, 15));
  });

  it('parses RFC3339 to UTC midnight', () => {
    expect(toUTCDate('2025-06-15T00:00:00Z')).toBe(Date.UTC(2025, 5, 15));
  });

  it('truncates time component to start of day', () => {
    // Even if a timestamp has non-zero time, toUTCDate returns start of that day
    expect(toUTCDate('2025-06-15T23:59:59Z')).toBe(Date.UTC(2025, 5, 15));
  });

  it('produces identical values for same date in different formats', () => {
    expect(toUTCDate('2025-06-15')).toBe(toUTCDate('2025-06-15T00:00:00Z'));
  });
});

// ---------------------------------------------------------------------------
// isActivePeriod
// ---------------------------------------------------------------------------
describe('isActivePeriod', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns true for period that spans today', () => {
    expect(isActivePeriod({ from: '2025-01-01', to: '2025-12-31' })).toBe(true);
  });

  it('returns false for future period', () => {
    expect(isActivePeriod({ from: '2025-09-01', to: '2025-12-31' })).toBe(false);
  });

  it('returns false for past period', () => {
    expect(isActivePeriod({ from: '2024-01-01', to: '2025-06-14' })).toBe(false);
  });

  it('returns true for RFC3339 dates on same day', () => {
    expect(isActivePeriod({ from: '2025-06-15T00:00:00Z', to: '2025-06-15T00:00:00Z' })).toBe(true);
  });

  it('returns true for open-ended period starting in the past', () => {
    expect(isActivePeriod({ from: '2025-01-01' })).toBe(true);
  });

  it('returns true for open-ended period starting today', () => {
    expect(isActivePeriod({ from: '2025-06-15' })).toBe(true);
  });

  it('returns false for open-ended period starting tomorrow', () => {
    expect(isActivePeriod({ from: '2025-06-16' })).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// getActiveContract
// ---------------------------------------------------------------------------
describe('getActiveContract', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns null for undefined', () => {
    expect(getActiveContract(undefined)).toBeNull();
  });

  it('returns null for empty array', () => {
    expect(getActiveContract([])).toBeNull();
  });

  it('returns active contract (no end date)', () => {
    const contracts = [{ from: '2025-01-01' }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('returns active contract (end date in future)', () => {
    const contracts = [{ from: '2025-01-01', to: '2025-12-31' }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('returns null when contract has not started yet', () => {
    const contracts = [{ from: '2025-09-01', to: '2025-12-31' }];
    expect(getActiveContract(contracts)).toBeNull();
  });

  it('returns null when contract has ended', () => {
    const contracts = [{ from: '2024-01-01', to: '2025-01-01' }];
    expect(getActiveContract(contracts)).toBeNull();
  });

  it('returns active contract among multiple', () => {
    const contracts = [
      { from: '2024-01-01', to: '2024-12-31' },
      { from: '2025-01-01', to: '2025-12-31' },
      { from: '2026-01-01', to: '2026-12-31' },
    ];
    expect(getActiveContract(contracts)).toBe(contracts[1]);
  });

  it('handles null to value', () => {
    const contracts = [{ from: '2025-01-01', to: null }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('handles RFC3339 from date (same day as today)', () => {
    const contracts = [{ from: '2025-06-15T00:00:00Z', to: null }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('handles RFC3339 from date (before today)', () => {
    const contracts = [{ from: '2025-06-14T00:00:00Z', to: null }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('handles RFC3339 to date (ended yesterday)', () => {
    const contracts = [{ from: '2025-01-01T00:00:00Z', to: '2025-06-14T00:00:00Z' }];
    expect(getActiveContract(contracts)).toBeNull();
  });

  it('handles RFC3339 to date (ends today)', () => {
    const contracts = [{ from: '2025-01-01T00:00:00Z', to: '2025-06-15T00:00:00Z' }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('handles mixed format: YYYY-MM-DD from with RFC3339 to', () => {
    const contracts = [{ from: '2025-01-01', to: '2025-12-31T00:00:00Z' }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });

  it('handles mixed format: RFC3339 from with YYYY-MM-DD to', () => {
    const contracts = [{ from: '2025-01-01T00:00:00Z', to: '2025-12-31' }];
    expect(getActiveContract(contracts)).toBe(contracts[0]);
  });
});

// ---------------------------------------------------------------------------
// getActiveContract - section transfer scenario
// ---------------------------------------------------------------------------
describe('getActiveContract - section transfer scenario', () => {
  // Reproduces the exact bug: after a section transfer, the old contract is
  // ended and a new contract is created. The employee should still be found.
  // Go backend serializes dates as RFC3339 ("2026-02-12T00:00:00Z").

  afterEach(() => {
    jest.useRealTimers();
  });

  it('finds active contract after section transfer (transfer yesterday)', () => {
    // Transfer happened on Feb 12. Today is Feb 13.
    // Old contract ended Feb 11 (yesterday of transfer day).
    // New contract started Feb 12.
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-02-13'));

    const contracts = [
      { from: '2025-01-01T00:00:00Z', to: '2026-02-11T00:00:00Z', staff_category: 'qualified' },
      { from: '2026-02-12T00:00:00Z', to: null, staff_category: 'qualified' },
    ];

    const active = getActiveContract(contracts);
    expect(active).toBe(contracts[1]);
    expect(getContractStatus(contracts[0])).toBe('ended');
    expect(getContractStatus(contracts[1])).toBe('active');
  });

  it('finds active contract after section transfer (transfer today)', () => {
    // Transfer happened today (Feb 13). Old contract ended Feb 12.
    // New contract starts today Feb 13.
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-02-13'));

    const contracts = [
      { from: '2025-01-01T00:00:00Z', to: '2026-02-12T00:00:00Z', staff_category: 'qualified' },
      { from: '2026-02-13T00:00:00Z', to: null, staff_category: 'qualified' },
    ];

    const active = getActiveContract(contracts);
    expect(active).toBe(contracts[1]);
    expect(getContractStatus(contracts[0])).toBe('ended');
    expect(getContractStatus(contracts[1])).toBe('active');
  });

  it('finds active contract among ended and upcoming (gap scenario)', () => {
    // An employee with a past contract, a current one, and a future one
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-06-15'));

    const contracts = [
      { from: '2025-01-01T00:00:00Z', to: '2025-12-31T00:00:00Z' },
      { from: '2026-01-01T00:00:00Z', to: '2026-12-31T00:00:00Z' },
      { from: '2027-01-01T00:00:00Z', to: null },
    ];

    expect(getActiveContract(contracts)).toBe(contracts[1]);
  });

  it('returns null when only ended and upcoming contracts exist (no active)', () => {
    // Gap between contracts: old ended yesterday, new starts tomorrow
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-02-13'));

    const contracts = [
      { from: '2025-01-01T00:00:00Z', to: '2026-02-12T00:00:00Z' },
      { from: '2026-02-14T00:00:00Z', to: null },
    ];

    expect(getActiveContract(contracts)).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// getCurrentContract
// ---------------------------------------------------------------------------
describe('getCurrentContract', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns null for undefined', () => {
    expect(getCurrentContract(undefined)).toBeNull();
  });

  it('returns null for empty array', () => {
    expect(getCurrentContract([])).toBeNull();
  });

  it('returns active contract when one exists', () => {
    const contracts = [{ from: '2025-01-01', to: '2025-12-31' }];
    expect(getCurrentContract(contracts)).toBe(contracts[0]);
  });

  it('falls back to contract with latest start date', () => {
    const contracts = [
      { from: '2023-01-01', to: '2023-12-31' },
      { from: '2024-06-01', to: '2024-12-31' },
      { from: '2024-01-01', to: '2024-06-30' },
    ];
    expect(getCurrentContract(contracts)).toEqual(contracts[1]);
  });

  it('prefers active contract over later-starting ended contract', () => {
    const contracts = [{ from: '2024-01-01', to: '2024-12-31' }, { from: '2025-01-01' }];
    expect(getCurrentContract(contracts)).toBe(contracts[1]);
  });

  it('does not mutate the original array when sorting', () => {
    const contracts = [
      { from: '2023-01-01', to: '2023-12-31' },
      { from: '2024-06-01', to: '2024-12-31' },
    ];
    const copy = [...contracts];
    getCurrentContract(contracts);
    expect(contracts).toEqual(copy);
  });

  it('falls back correctly with RFC3339 dates', () => {
    const contracts = [
      { from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
      { from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
    ];
    // Neither is active (today is 2025-06-15), falls back to latest start date
    expect(getCurrentContract(contracts)).toEqual(contracts[1]);
  });
});

// ---------------------------------------------------------------------------
// getDayBefore
// ---------------------------------------------------------------------------
describe('getDayBefore', () => {
  it('returns day before a date', () => {
    expect(getDayBefore('2025-06-15')).toBe('2025-06-14');
  });

  it('crosses month boundary', () => {
    expect(getDayBefore('2025-03-01')).toBe('2025-02-28');
  });

  it('crosses year boundary', () => {
    expect(getDayBefore('2025-01-01')).toBe('2024-12-31');
  });

  it('handles leap year', () => {
    expect(getDayBefore('2024-03-01')).toBe('2024-02-29');
  });
});

// ---------------------------------------------------------------------------
// getContractStatus
// ---------------------------------------------------------------------------
describe('getContractStatus', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns null for null contract', () => {
    expect(getContractStatus(null)).toBeNull();
  });

  it('returns active for current contract (no end date)', () => {
    expect(getContractStatus({ from: '2025-01-01' })).toBe('active');
  });

  it('returns active for current contract (end date in future)', () => {
    expect(getContractStatus({ from: '2025-01-01', to: '2025-12-31' })).toBe('active');
  });

  it('returns upcoming for future contract', () => {
    expect(getContractStatus({ from: '2025-09-01' })).toBe('upcoming');
  });

  it('returns ended for past contract', () => {
    expect(getContractStatus({ from: '2024-01-01', to: '2025-01-01' })).toBe('ended');
  });

  it('returns active for contract ending today', () => {
    expect(getContractStatus({ from: '2025-01-01', to: '2025-06-15' })).toBe('active');
  });

  it('returns active for contract starting today', () => {
    expect(getContractStatus({ from: '2025-06-15' })).toBe('active');
  });

  it('handles null to value', () => {
    expect(getContractStatus({ from: '2025-01-01', to: null })).toBe('active');
  });

  it('returns active for RFC3339 from date starting today', () => {
    expect(getContractStatus({ from: '2025-06-15T00:00:00Z' })).toBe('active');
  });

  it('returns upcoming for RFC3339 from date in future', () => {
    expect(getContractStatus({ from: '2025-09-01T00:00:00Z' })).toBe('upcoming');
  });

  it('returns ended for RFC3339 to date ended yesterday', () => {
    expect(getContractStatus({ from: '2024-01-01T00:00:00Z', to: '2025-06-14T00:00:00Z' })).toBe(
      'ended'
    );
  });

  it('returns active for RFC3339 to date ending today', () => {
    expect(getContractStatus({ from: '2024-01-01T00:00:00Z', to: '2025-06-15T00:00:00Z' })).toBe(
      'active'
    );
  });

  it('returns active for contract starting today with RFC3339 (original bug scenario)', () => {
    // This is the exact scenario that caused the original bug:
    // Go serializes "2025-06-15" as "2025-06-15T00:00:00Z", and the old
    // string comparison "2025-06-15T00:00:00Z" > "2025-06-15" was true,
    // incorrectly returning "upcoming".
    expect(getContractStatus({ from: '2025-06-15T00:00:00Z' })).toBe('active');
    expect(getContractStatus({ from: '2025-06-15T00:00:00Z', to: null })).toBe('active');
  });

  it('returns ended for contract that ended today-1 in RFC3339', () => {
    // Ended yesterday: to = 2025-06-14T00:00:00Z
    expect(getContractStatus({ from: '2025-01-01T00:00:00Z', to: '2025-06-14T00:00:00Z' })).toBe(
      'ended'
    );
  });

  it('returns upcoming for contract starting tomorrow in RFC3339', () => {
    expect(getContractStatus({ from: '2025-06-16T00:00:00Z' })).toBe('upcoming');
  });
});

// ---------------------------------------------------------------------------
// compareDates
// ---------------------------------------------------------------------------
describe('compareDates', () => {
  it('returns negative when a is before b', () => {
    expect(compareDates('2025-01-01', '2025-06-15')).toBeLessThan(0);
  });

  it('returns positive when a is after b', () => {
    expect(compareDates('2025-12-31', '2025-06-15')).toBeGreaterThan(0);
  });

  it('returns 0 for same dates', () => {
    expect(compareDates('2025-06-15', '2025-06-15')).toBe(0);
  });

  it('returns 0 for same date in different formats', () => {
    expect(compareDates('2025-06-15', '2025-06-15T00:00:00Z')).toBe(0);
  });

  it('works correctly for sorting', () => {
    const dates = ['2025-03-01', '2025-01-01T00:00:00Z', '2025-02-01'];
    const sorted = [...dates].sort(compareDates);
    expect(sorted).toEqual(['2025-01-01T00:00:00Z', '2025-02-01', '2025-03-01']);
  });
});

// ---------------------------------------------------------------------------
// isDateBefore
// ---------------------------------------------------------------------------
describe('isDateBefore', () => {
  it('returns true when a is before b', () => {
    expect(isDateBefore('2025-01-01', '2025-06-15')).toBe(true);
  });

  it('returns false when a is after b', () => {
    expect(isDateBefore('2025-12-31', '2025-06-15')).toBe(false);
  });

  it('returns false for same dates', () => {
    expect(isDateBefore('2025-06-15', '2025-06-15')).toBe(false);
  });

  it('returns false for same date in different formats', () => {
    expect(isDateBefore('2025-06-15', '2025-06-15T00:00:00Z')).toBe(false);
  });

  it('works with RFC3339 dates', () => {
    expect(isDateBefore('2025-01-01T00:00:00Z', '2025-06-15T00:00:00Z')).toBe(true);
    expect(isDateBefore('2025-06-15T00:00:00Z', '2025-01-01T00:00:00Z')).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Edge cases: boundary dates and format consistency
// ---------------------------------------------------------------------------
describe('edge cases', () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  it('contract starting and ending on same day is active on that day', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
    expect(getContractStatus({ from: '2025-06-15', to: '2025-06-15' })).toBe('active');
    expect(getContractStatus({ from: '2025-06-15T00:00:00Z', to: '2025-06-15T00:00:00Z' })).toBe(
      'active'
    );
  });

  it('contract ending yesterday is ended, not active', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
    expect(getContractStatus({ from: '2025-01-01', to: '2025-06-14' })).toBe('ended');
  });

  it('contract starting tomorrow is upcoming, not active', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
    expect(getContractStatus({ from: '2025-06-16' })).toBe('upcoming');
  });

  it('year boundary: Dec 31 to Jan 1 transition', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-12-31'));
    // Contract ending today
    expect(getContractStatus({ from: '2025-01-01', to: '2025-12-31' })).toBe('active');
    // Contract starting tomorrow
    expect(getContractStatus({ from: '2026-01-01' })).toBe('upcoming');
  });

  it('handles consecutive contracts without gap (transfer scenario)', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-02-13'));

    // Old contract: Jan 1 2025 to Feb 11 2026
    // New contract: Feb 12 2026 onwards (no end date)
    // Today: Feb 13 2026
    // Both dates are RFC3339 (as Go would serialize them)
    const contracts = [
      { from: '2025-01-01T00:00:00Z', to: '2026-02-11T00:00:00Z' },
      { from: '2026-02-12T00:00:00Z', to: null },
    ];

    const active = getActiveContract(contracts);
    expect(active).not.toBeNull();
    expect(active).toBe(contracts[1]);

    // getCurrentContract should also find the active one
    const current = getCurrentContract(contracts);
    expect(current).toBe(contracts[1]);
  });

  it('handles new Date() with and without T suffix identically', () => {
    // This is the core of the original bug: ensure "2025-06-15" and
    // "2025-06-15T00:00:00Z" are treated as the same date
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));

    const withT = { from: '2025-06-15T00:00:00Z', to: '2025-12-31T00:00:00Z' };
    const withoutT = { from: '2025-06-15', to: '2025-12-31' };

    expect(getContractStatus(withT)).toBe(getContractStatus(withoutT));
    expect(isActivePeriod(withT)).toBe(isActivePeriod(withoutT));
  });
});
