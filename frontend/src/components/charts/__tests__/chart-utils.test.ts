import {
  kitaYearLabel,
  buildKitaYearBands,
  formatDateLabel,
  createTodayMarker,
} from '../chart-utils';

// ============================================================
// kitaYearLabel
// ============================================================

describe('kitaYearLabel', () => {
  it('returns correct label for August (start of new Kita year)', () => {
    expect(kitaYearLabel('2024-08-01')).toBe('24/25');
  });

  it('returns correct label for July (last month of previous Kita year)', () => {
    expect(kitaYearLabel('2025-07-01')).toBe('24/25');
  });

  it('returns correct label for January (mid Kita year)', () => {
    expect(kitaYearLabel('2025-01-01')).toBe('24/25');
  });

  it('returns correct label for December', () => {
    expect(kitaYearLabel('2024-12-01')).toBe('24/25');
  });

  it('returns correct label for September', () => {
    expect(kitaYearLabel('2024-09-01')).toBe('24/25');
  });

  it('handles year boundary correctly: Jan 2024 belongs to 23/24', () => {
    expect(kitaYearLabel('2024-01-01')).toBe('23/24');
  });

  it('handles year boundary correctly: Jul 2024 belongs to 23/24', () => {
    expect(kitaYearLabel('2024-07-01')).toBe('23/24');
  });

  it('handles year boundary correctly: Aug 2024 belongs to 24/25', () => {
    expect(kitaYearLabel('2024-08-01')).toBe('24/25');
  });

  it('handles century boundary (year 2099-08 -> 99/00)', () => {
    expect(kitaYearLabel('2099-08-01')).toBe('99/00');
  });

  it('handles century boundary (year 2100-01 -> 99/00)', () => {
    expect(kitaYearLabel('2100-01-01')).toBe('99/00');
  });

  it('works with mid-month dates', () => {
    expect(kitaYearLabel('2024-08-15')).toBe('24/25');
    expect(kitaYearLabel('2025-03-20')).toBe('24/25');
  });
});

// ============================================================
// buildKitaYearBands
// ============================================================

describe('buildKitaYearBands', () => {
  it('returns empty array for empty input', () => {
    expect(buildKitaYearBands([])).toEqual([]);
  });

  it('returns single band for one date', () => {
    const result = buildKitaYearBands(['2024-09-01']);
    expect(result).toEqual([{ label: '24/25', startIdx: 0, endIdx: 0 }]);
  });

  it('returns single band for dates within same Kita year', () => {
    const dates = ['2024-09-01', '2024-10-01', '2024-11-01', '2024-12-01'];
    const result = buildKitaYearBands(dates);
    expect(result).toEqual([{ label: '24/25', startIdx: 0, endIdx: 3 }]);
  });

  it('splits at Aug boundary into two bands', () => {
    const dates = [
      '2024-06-01', // 23/24
      '2024-07-01', // 23/24
      '2024-08-01', // 24/25
      '2024-09-01', // 24/25
    ];
    const result = buildKitaYearBands(dates);
    expect(result).toEqual([
      { label: '23/24', startIdx: 0, endIdx: 1 },
      { label: '24/25', startIdx: 2, endIdx: 3 },
    ]);
  });

  it('handles three Kita years spanning 24 months', () => {
    const dates = [
      '2023-08-01', // 23/24
      '2024-01-01', // 23/24
      '2024-07-01', // 23/24
      '2024-08-01', // 24/25
      '2025-01-01', // 24/25
      '2025-07-01', // 24/25
      '2025-08-01', // 25/26
    ];
    const result = buildKitaYearBands(dates);
    expect(result).toEqual([
      { label: '23/24', startIdx: 0, endIdx: 2 },
      { label: '24/25', startIdx: 3, endIdx: 5 },
      { label: '25/26', startIdx: 6, endIdx: 6 },
    ]);
  });

  it('handles full 12-month range within one Kita year (Aug-Jul)', () => {
    const dates = [
      '2024-08-01',
      '2024-09-01',
      '2024-10-01',
      '2024-11-01',
      '2024-12-01',
      '2025-01-01',
      '2025-02-01',
      '2025-03-01',
      '2025-04-01',
      '2025-05-01',
      '2025-06-01',
      '2025-07-01',
    ];
    const result = buildKitaYearBands(dates);
    expect(result).toEqual([{ label: '24/25', startIdx: 0, endIdx: 11 }]);
  });

  it('handles consecutive single-month bands at boundary', () => {
    // Jul then Aug — each one month in a different Kita year
    const dates = ['2024-07-01', '2024-08-01'];
    const result = buildKitaYearBands(dates);
    expect(result).toEqual([
      { label: '23/24', startIdx: 0, endIdx: 0 },
      { label: '24/25', startIdx: 1, endIdx: 1 },
    ]);
  });
});

// ============================================================
// formatDateLabel
// ============================================================

describe('formatDateLabel', () => {
  it('formats January correctly', () => {
    expect(formatDateLabel('2025-01-01')).toBe('Jan 25');
  });

  it('formats August correctly', () => {
    expect(formatDateLabel('2024-08-01')).toBe('Aug 24');
  });

  it('formats December correctly', () => {
    expect(formatDateLabel('2024-12-01')).toBe('Dec 24');
  });

  it('ignores day portion', () => {
    expect(formatDateLabel('2025-03-15')).toBe('Mar 25');
  });

  it('formats year 2000 correctly', () => {
    expect(formatDateLabel('2000-06-01')).toBe('Jun 00');
  });
});

// ============================================================
// createTodayMarker
// ============================================================

describe('createTodayMarker', () => {
  it('returns marker config with correct value and legend', () => {
    const marker = createTodayMarker('Feb 25', 'Today');
    expect(marker.axis).toBe('x');
    expect(marker.value).toBe('Feb 25');
    expect(marker.legend).toBe('Today');
    expect(marker.legendPosition).toBe('top');
  });

  it('includes dashed line style', () => {
    const marker = createTodayMarker('Jan 24', 'Heute');
    expect(marker.lineStyle.strokeDasharray).toBe('4 4');
    expect(marker.lineStyle.strokeWidth).toBe(1);
  });
});
