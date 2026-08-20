import {
  calculateContractEndDate,
  classifySchoolEnrollment,
  suggestContractEnd,
} from '../school-enrollment';

describe('calculateContractEndDate', () => {
  describe('Berlin (Stichtag: September 30)', () => {
    // Child born Jan 15, 2020 → turns 6 on Jan 15, 2026 (before Sep 30, 2026)
    // → starts school Aug 2026 → Kita ends July 31, 2026
    it('returns July 31 of the year child turns 6 when birthday is before Stichtag', () => {
      expect(calculateContractEndDate('2020-01-15', 'berlin')).toBe('2026-07-31');
    });

    // Child born Sep 30, 2020 → turns 6 on Sep 30, 2026 (on Stichtag)
    // → starts school Aug 2026 → Kita ends July 31, 2026
    it('returns July 31 of the same year when birthday is on Stichtag', () => {
      expect(calculateContractEndDate('2020-09-30', 'berlin')).toBe('2026-07-31');
    });

    // Child born Oct 1, 2020 → turns 6 on Oct 1, 2026 (after Sep 30, 2026)
    // → starts school Aug 2027 → Kita ends July 31, 2027
    it('returns July 31 of the next year when birthday is after Stichtag', () => {
      expect(calculateContractEndDate('2020-10-01', 'berlin')).toBe('2027-07-31');
    });

    // Child born Dec 31, 2019 → turns 6 on Dec 31, 2025 (after Sep 30, 2025)
    // → starts school Aug 2026 → Kita ends July 31, 2026
    it('handles late-year birthdays correctly', () => {
      expect(calculateContractEndDate('2019-12-31', 'berlin')).toBe('2026-07-31');
    });

    // Child born Jul 31, 2021 → turns 6 on Jul 31, 2027 (before Sep 30, 2027)
    // → starts school Aug 2027 → Kita ends July 31, 2027
    it('handles mid-year birthdays correctly', () => {
      expect(calculateContractEndDate('2021-07-31', 'berlin')).toBe('2027-07-31');
    });
  });

  describe('edge cases', () => {
    it('returns null for empty birthdate', () => {
      expect(calculateContractEndDate('', 'berlin')).toBeNull();
    });

    it('returns null for empty state', () => {
      expect(calculateContractEndDate('2020-01-15', '')).toBeNull();
    });

    it('returns null for invalid birthdate', () => {
      expect(calculateContractEndDate('not-a-date', 'berlin')).toBeNull();
    });

    it('uses default Stichtag for unknown state', () => {
      // Default is same as Berlin (Sep 30), so same result
      expect(calculateContractEndDate('2020-01-15', 'unknown-state')).toBe('2026-07-31');
    });
  });
});

describe('classifySchoolEnrollment', () => {
  describe('Berlin — Muss-only (sixth birthday in Apr–Sep)', () => {
    // Child born Jun 1, 2020 → turns 6 Jun 1, 2026 (Apr–Sep 2026 → no Kann window match)
    it('classifies an April–September birthday as Muss-only', () => {
      expect(classifySchoolEnrollment('2020-06-01', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2026,
        mussYear: 2026,
        kannYear: null,
        mussContractEnd: '2026-07-31',
        kannContractEnd: null,
      });
    });

    it('classifies Sep 30 (on Stichtag) as Muss-only', () => {
      expect(classifySchoolEnrollment('2020-09-30', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2026,
        mussYear: 2026,
        kannYear: null,
        mussContractEnd: '2026-07-31',
        kannContractEnd: null,
      });
    });

    it('classifies Apr 1 as Muss-only (one day after Kann window closes)', () => {
      expect(classifySchoolEnrollment('2021-04-01', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2027,
        mussYear: 2027,
        kannYear: null,
        mussContractEnd: '2027-07-31',
        kannContractEnd: null,
      });
    });
  });

  describe('Berlin — Kann-eligible (sixth birthday in Oct–Mar window)', () => {
    // Per Berlin rule, the Kann option is for the August *preceding* the child's
    // sixth birthday. A January-born child is Muss the year they turn 6 and
    // also Kann the August before (at age 5½).
    it('classifies a January birthday (before Stichtag) as Muss and Kann', () => {
      expect(classifySchoolEnrollment('2020-01-15', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2026,
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });

    it('classifies Oct 1 as Kann-eligible (window start)', () => {
      expect(classifySchoolEnrollment('2020-10-01', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2027,
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies a late-autumn birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2020-11-15', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2027,
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies a Dec 31 birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2019-12-31', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2026,
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });

    it('classifies a February birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2021-02-15', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2027,
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies Mar 31 as Kann-eligible (window end, inclusive)', () => {
      expect(classifySchoolEnrollment('2021-03-31', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2027,
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });
  });

  describe('edge cases', () => {
    it('returns null for empty birthdate', () => {
      expect(classifySchoolEnrollment('', 'berlin')).toBeNull();
    });

    it('returns null for empty state', () => {
      expect(classifySchoolEnrollment('2020-01-15', '')).toBeNull();
    });

    it('returns null for invalid birthdate', () => {
      expect(classifySchoolEnrollment('not-a-date', 'berlin')).toBeNull();
    });

    it('returns null for unknown state (strict)', () => {
      expect(classifySchoolEnrollment('2020-01-15', 'bayern')).toBeNull();
    });

    // The API returns birthdate as a full ISO datetime; the classifier has to
    // accept that shape in addition to the form's "YYYY-MM-DD" input.
    it('accepts an ISO datetime birthdate (as returned by the API)', () => {
      expect(classifySchoolEnrollment('2020-01-15T00:00:00Z', 'berlin')).toEqual({
        overridden: false,
        computedMussYear: 2026,
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });
  });
});

describe('suggestContractEnd', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    // A child born 2020-08-20 turns 6 before the 30 Sep Stichtag, so their
    // muss school year is 2026 and their contract end 2026-07-31 -- three
    // weeks in the past on this date.
    jest.setSystemTime(new Date('2026-08-20T09:00:00Z'));
  });
  afterEach(() => {
    jest.useRealTimers();
  });

  it('suggests the muss end date while it is still ahead', () => {
    expect(suggestContractEnd('2021-08-20', 'berlin')).toBe('2027-07-31');
  });

  it('suggests nothing once the muss end has passed', () => {
    expect(suggestContractEnd('2020-08-20', 'berlin')).toBe('');
  });

  it('suggests nothing when the end would precede the start it sits next to', () => {
    // The dialog's own case: it opens with from = tomorrow.
    expect(suggestContractEnd('2020-08-20', 'berlin', '2026-08-21')).toBe('');
  });

  it('measures against the given start, not today', () => {
    // Backdated entry: a start in 2025 makes a 2026-07-31 end perfectly valid,
    // even though that date is behind us now.
    expect(suggestContractEnd('2020-08-20', 'berlin', '2025-09-01')).toBe('2026-07-31');
  });

  it('does not suggest an end equal to the day before the start', () => {
    // The one-day case a "is it in the past" test would miss: the end is not
    // past, but it still precedes the start.
    jest.setSystemTime(new Date('2026-07-31T09:00:00Z'));
    expect(suggestContractEnd('2020-08-20', 'berlin', '2026-08-01')).toBe('');
  });

  it('keeps an end that falls exactly on the start', () => {
    jest.setSystemTime(new Date('2026-07-01T09:00:00Z'));
    expect(suggestContractEnd('2020-08-20', 'berlin', '2026-07-31')).toBe('2026-07-31');
  });

  it('suggests nothing for an unknown state or unusable birthdate', () => {
    expect(suggestContractEnd('2021-08-20', 'bayern')).toBe('2027-07-31'); // lenient fallback
    expect(suggestContractEnd('', 'berlin')).toBe('');
    expect(suggestContractEnd('2021-08-20', '')).toBe('');
  });
});

describe('a recorded school entry date (Zurückstellung)', () => {
  // Born 2020-05-15: six before the 30 Sep Stichtag, so the birthdate alone
  // says school in 2026 and Kita until 2026-07-31.
  const birthdate = '2020-05-15';

  it('overrides the computed year and end date', () => {
    const e = classifySchoolEnrollment(birthdate, 'berlin', '2027-08-01');
    expect(e).not.toBeNull();
    expect(e!.overridden).toBe(true);
    expect(e!.mussYear).toBe(2027);
    expect(e!.mussContractEnd).toBe('2027-07-31');
  });

  it('keeps the computed year so a caller can say which way it diverges', () => {
    const deferred = classifySchoolEnrollment(birthdate, 'berlin', '2027-08-01');
    expect(deferred!.computedMussYear).toBe(2026);
    expect(deferred!.mussYear).toBeGreaterThan(deferred!.computedMussYear!);

    const early = classifySchoolEnrollment(birthdate, 'berlin', '2025-08-01');
    expect(early!.mussYear).toBeLessThan(early!.computedMussYear!);
  });

  it('drops the Kann alternative once a decision is on file', () => {
    const e = classifySchoolEnrollment(birthdate, 'berlin', '2027-08-01');
    expect(e!.kannYear).toBeNull();
    expect(e!.kannContractEnd).toBeNull();
  });

  it('answers even for a state with no rules', () => {
    // Knowing the date does not depend on being able to derive it.
    const e = classifySchoolEnrollment(birthdate, 'bayern', '2027-08-01');
    expect(e).not.toBeNull();
    expect(e!.mussContractEnd).toBe('2027-07-31');
    expect(e!.computedMussYear).toBeNull();
  });

  it('accepts the API date-time form, not just YYYY-MM-DD', () => {
    const e = classifySchoolEnrollment(birthdate, 'berlin', '2027-08-01T00:00:00Z');
    expect(e!.mussContractEnd).toBe('2027-07-31');
  });

  it('falls back to the computed values when no date is recorded', () => {
    const e = classifySchoolEnrollment(birthdate, 'berlin');
    expect(e!.overridden).toBe(false);
    expect(e!.mussYear).toBe(2026);
    expect(e!.computedMussYear).toBe(2026);
  });

  it('drives the contract-end suggestion too', () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-08-20T09:00:00Z'));
    // Without the recorded date this child is past their computed end and gets
    // no suggestion at all; with it, the deferred year is offered.
    expect(suggestContractEnd(birthdate, 'berlin', '2026-08-21')).toBe('');
    expect(suggestContractEnd(birthdate, 'berlin', '2026-08-21', '2027-08-01')).toBe('2027-07-31');
    jest.useRealTimers();
  });
});
