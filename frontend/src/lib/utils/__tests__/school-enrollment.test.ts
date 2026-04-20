import { calculateContractEndDate, classifySchoolEnrollment } from '../school-enrollment';

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
        mussYear: 2026,
        kannYear: null,
        mussContractEnd: '2026-07-31',
        kannContractEnd: null,
      });
    });

    it('classifies Sep 30 (on Stichtag) as Muss-only', () => {
      expect(classifySchoolEnrollment('2020-09-30', 'berlin')).toEqual({
        mussYear: 2026,
        kannYear: null,
        mussContractEnd: '2026-07-31',
        kannContractEnd: null,
      });
    });

    it('classifies Apr 1 as Muss-only (one day after Kann window closes)', () => {
      expect(classifySchoolEnrollment('2021-04-01', 'berlin')).toEqual({
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
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });

    it('classifies Oct 1 as Kann-eligible (window start)', () => {
      expect(classifySchoolEnrollment('2020-10-01', 'berlin')).toEqual({
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies a late-autumn birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2020-11-15', 'berlin')).toEqual({
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies a Dec 31 birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2019-12-31', 'berlin')).toEqual({
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });

    it('classifies a February birthday as Kann-eligible', () => {
      expect(classifySchoolEnrollment('2021-02-15', 'berlin')).toEqual({
        mussYear: 2027,
        kannYear: 2026,
        mussContractEnd: '2027-07-31',
        kannContractEnd: '2026-07-31',
      });
    });

    it('classifies Mar 31 as Kann-eligible (window end, inclusive)', () => {
      expect(classifySchoolEnrollment('2021-03-31', 'berlin')).toEqual({
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
        mussYear: 2026,
        kannYear: 2025,
        mussContractEnd: '2026-07-31',
        kannContractEnd: '2025-07-31',
      });
    });
  });
});
