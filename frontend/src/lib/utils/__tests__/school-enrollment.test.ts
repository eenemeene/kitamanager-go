import { calculateContractEndDate } from '../school-enrollment';

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
