jest.mock('@nivo/bar', () => ({ ResponsiveBar: () => null }));

// eslint-disable-next-line import/first
import {
  buildSectionStaffingRows,
  computeSymmetricDomainMax,
  type SectionStaffingData,
} from '../section-staffing-chart';

const KEY = 'pct'; // Tests use a fixed key so they're locale-agnostic.

describe('buildSectionStaffingRows', () => {
  describe('per-section percentage math', () => {
    it('returns +10% when available exceeds required by 10%', () => {
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'Krippe', required: 100, available: 110 }],
        KEY
      );
      expect(rows[0]).toMatchObject({
        section: 'Krippe',
        [KEY]: 10,
        available: 110,
        required: 100,
      });
    });

    it('returns -10% when available is below required by 10%', () => {
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'Krippe', required: 100, available: 90 }],
        KEY
      );
      expect(rows[0]?.[KEY]).toBe(-10);
    });

    it('returns 0 when required is 0 (no division by zero)', () => {
      // Section with no required hours (e.g. closed for the period)
      // should render as 0% — not Infinity, NaN, or crash the chart.
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'Closed', required: 0, available: 50 }],
        KEY
      );
      expect(rows[0]?.[KEY]).toBe(0);
    });

    it('returns 0 when available exactly equals required', () => {
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'Bal', required: 25.5, available: 25.5 }],
        KEY
      );
      expect(rows[0]?.[KEY]).toBe(0);
    });

    it('rounds the percentage to one decimal', () => {
      // (109 - 105) / 105 * 100 = 3.80952... → 3.8
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'F', required: 105, available: 109 }],
        KEY
      );
      expect(rows[0]?.[KEY]).toBe(3.8);
    });
  });

  describe('hour rounding', () => {
    it('rounds available and required hours to whole numbers (axis tick formatter expects integers)', () => {
      // Y-axis labels read "Section (108h / 100h)" — fractional hours
      // would clutter the tick labels. The component uses these rounded
      // values for both display and tooltips.
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'X', required: 100.4, available: 108.6 }],
        KEY
      );
      expect(rows[0]?.required).toBe(100);
      expect(rows[0]?.available).toBe(109);
    });

    it('rounds .5 boundary using banker-free Math.round (away from zero)', () => {
      // 99.5 → 100, 100.5 → 101 (Math.round rounds toward +Infinity at .5)
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'B', required: 99.5, available: 100.5 }],
        KEY
      );
      expect(rows[0]?.required).toBe(100);
      expect(rows[0]?.available).toBe(101);
    });
  });

  describe('balanceKey injection', () => {
    it('uses the supplied key (not a literal) so the legend can localise', () => {
      const rows = buildSectionStaffingRows(
        [{ sectionName: 'X', required: 1, available: 1 }],
        'Bilanz %'
      );
      expect('Bilanz %' in rows[0]!).toBe(true);
      expect((rows[0] as Record<string, number>)['Bilanz %']).toBe(0);
    });
  });

  describe('input shape', () => {
    it('returns one row per input, in order', () => {
      const data: SectionStaffingData[] = [
        { sectionName: 'A', required: 100, available: 100 },
        { sectionName: 'B', required: 100, available: 110 },
        { sectionName: 'C', required: 100, available: 80 },
      ];
      const rows = buildSectionStaffingRows(data, KEY);
      expect(rows.map((r) => r.section)).toEqual(['A', 'B', 'C']);
      expect(rows.map((r) => r[KEY])).toEqual([0, 10, -20]);
    });

    it('returns [] for empty input', () => {
      expect(buildSectionStaffingRows([], KEY)).toEqual([]);
    });

    it('does not mutate the input array', () => {
      const data: SectionStaffingData[] = [{ sectionName: 'A', required: 50, available: 60 }];
      const before = JSON.stringify(data);
      buildSectionStaffingRows(data, KEY);
      expect(JSON.stringify(data)).toBe(before);
    });
  });
});

describe('computeSymmetricDomainMax', () => {
  it('floors the domain at 10 even when all percentages are tiny', () => {
    // Symmetric domain → if every section is at ±2%, the chart still
    // renders with a ±11 (10 + 10% padding → 11) domain so the user
    // sees "this is fine" and small bars don't fill the chart visually.
    expect(computeSymmetricDomainMax([1, -2, 3])).toBe(11);
  });

  it('expands above 10 when a percentage exceeds it (with ~10% padding)', () => {
    // max(|...|) = 25 → 25 * 1.1 = 27.5 → ceil → 28
    expect(computeSymmetricDomainMax([10, -25, 5])).toBe(28);
  });

  it('uses the largest absolute value (negative or positive)', () => {
    // The y-axis is symmetric — a -50% bar on one side needs the same
    // upper bound on the other side for consistent grid spacing.
    // 50 * 1.1 = 55.00000000000001 in IEEE-754, so Math.ceil bumps to
    // 56. Slightly more padding than nominal — still correct intent.
    expect(computeSymmetricDomainMax([5, -50])).toBe(56);
  });

  it('returns 11 for empty input (default 10 + 10% padding)', () => {
    // Defensive: an empty data array shouldn't crash the chart with a
    // collapsed domain. Math.max(...[]) is -Infinity, but Math.max(10)
    // saves us.
    expect(computeSymmetricDomainMax([])).toBe(11);
  });

  it('rounds up to the next integer (ceil) so labels stay whole', () => {
    // 12 * 1.1 = 13.2 → 14
    expect(computeSymmetricDomainMax([12])).toBe(14);
  });
});
