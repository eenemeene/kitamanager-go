// Stub Nivo + d3-scale so importing the chart module doesn't pull d3
// ESM through. The transform under test never touches them.
jest.mock('@nivo/line', () => ({ ResponsiveLine: () => null }));
jest.mock('d3-scale', () => ({ scaleLinear: () => () => 0 }));

import { computeBalancePercentages } from '../staffing-hours-chart';
import type { StaffingHoursResponse } from '@/lib/api/types';

type DataPoint = StaffingHoursResponse['data_points'][number];

function makeDp(overrides: Partial<DataPoint> = {}): DataPoint {
  return {
    date: '2025-01-01',
    required_hours: 0,
    available_hours: 0,
    child_count: 0,
    staff_count: 0,
    ...overrides,
  };
}

describe('computeBalancePercentages', () => {
  describe('no requirement means no ratio', () => {
    // These returned 0 and the table beside the chart returned a dash, both
    // written in 4fd044f0 -- one commit, two answers to the same question,
    // because that commit was pinning behaviour rather than choosing it.
    //
    // The dash is the true one. (available - required) / required is undefined
    // at required = 0, and 0% positively asserts "exactly staffed" for a month
    // with staff on shift and no children. Infinity must still never reach the
    // d3 scale, which is what the original guard was really protecting; NaN is
    // filtered before the domain is computed and skipped when drawing.
    it('returns NaN when required_hours is 0 (does not divide)', () => {
      const res = computeBalancePercentages([makeDp({ required_hours: 0, available_hours: 100 })]);
      expect(res).toHaveLength(1);
      expect(Number.isFinite(res[0])).toBe(false);
    });

    it('returns NaN when required_hours is undefined (older payload)', () => {
      const dp = { date: '2025-01-01' } as DataPoint;
      expect(Number.isFinite(computeBalancePercentages([dp])[0])).toBe(false);
    });

    it('returns NaN when both required and available are undefined', () => {
      const dp = { date: '2025-01-01' } as DataPoint;
      expect(Number.isFinite(computeBalancePercentages([dp])[0])).toBe(false);
    });

    it('never returns Infinity, which is what would break the scale', () => {
      const res = computeBalancePercentages([makeDp({ required_hours: 0, available_hours: 999 })]);
      expect(res[0]).not.toBe(Infinity);
      expect(res[0]).not.toBe(-Infinity);
    });
  });

  describe('happy paths', () => {
    it('returns positive percentage when available exceeds required', () => {
      // 110 / 100 = 1.10 → +10%
      const res = computeBalancePercentages([
        makeDp({ required_hours: 100, available_hours: 110 }),
      ]);
      expect(res).toEqual([10]);
    });

    it('returns negative percentage when available is below required', () => {
      // 90 / 100 = 0.9 → -10%
      const res = computeBalancePercentages([makeDp({ required_hours: 100, available_hours: 90 })]);
      expect(res).toEqual([-10]);
    });

    it('returns 0 when available exactly equals required', () => {
      const res = computeBalancePercentages([
        makeDp({ required_hours: 50.5, available_hours: 50.5 }),
      ]);
      expect(res).toEqual([0]);
    });
  });

  describe('rounding to 0.1%', () => {
    it('rounds non-trivial fractions to one decimal', () => {
      // (108 - 100) / 100 * 100 = 8 — clean.
      // Tweak: 33/300 = 0.11 → +11%. Try a real fraction:
      // (107 - 100) / 100 * 1000 / 10 = 7
      // (109 - 105) / 105 = 0.0381 → 3.8 (not 3.80952)
      const res = computeBalancePercentages([
        makeDp({ required_hours: 105, available_hours: 109 }),
      ]);
      expect(res).toEqual([3.8]);
    });

    it('does NOT round to whole percent (would lose detail at small variances)', () => {
      // Defensive: Math.round(((1) / 100) * 1000) / 10 = 1.0, not 1
      // — must keep the .toFixed-style decimal.
      // 101 / 100 = 1.01 → +1.0%
      const res = computeBalancePercentages([
        makeDp({ required_hours: 100, available_hours: 101 }),
      ]);
      expect(res).toEqual([1]);
    });

    it('handles negative fractions consistently', () => {
      // (105 - 109) / 109 * 100 = -3.6697... → rounds to -3.7
      const res = computeBalancePercentages([
        makeDp({ required_hours: 109, available_hours: 105 }),
      ]);
      expect(res).toEqual([-3.7]);
    });
  });

  describe('input shape', () => {
    it('returns one entry per data point, in order', () => {
      const res = computeBalancePercentages([
        makeDp({ required_hours: 100, available_hours: 100 }),
        makeDp({ required_hours: 100, available_hours: 110 }),
        makeDp({ required_hours: 100, available_hours: 80 }),
      ]);
      expect(res).toEqual([0, 10, -20]);
    });

    it('returns [] for empty data_points', () => {
      expect(computeBalancePercentages([])).toEqual([]);
    });

    it('does not mutate the input array', () => {
      const data = [makeDp({ required_hours: 50, available_hours: 60 })];
      const before = JSON.stringify(data);
      computeBalancePercentages(data);
      expect(JSON.stringify(data)).toBe(before);
    });
  });

  describe('extreme values', () => {
    it('handles enormous available against tiny required without exploding', () => {
      // (10_000 - 1) / 1 = 9999 → 999900%? Actually let's compute:
      // (10000 - 1) / 1 * 1000 = 9_999_000 → /10 = 999_900
      // The chart's right axis would adapt; the function shouldn't cap.
      const res = computeBalancePercentages([
        makeDp({ required_hours: 1, available_hours: 10_000 }),
      ]);
      expect(res).toEqual([999_900]);
    });

    it('handles fractional required_hours (part-time week)', () => {
      // 19.5h required, 20h available → +2.6%
      const res = computeBalancePercentages([
        makeDp({ required_hours: 19.5, available_hours: 20 }),
      ]);
      expect(res).toEqual([2.6]);
    });
  });
});
