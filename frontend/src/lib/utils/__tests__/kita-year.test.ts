import { buildKitaYearCompareWindows, kitaYearLabel } from '../kita-year';

describe('kitaYearLabel', () => {
  it('starts a new Kita year in August', () => {
    expect(kitaYearLabel('2025-07-01')).toBe('24/25');
    expect(kitaYearLabel('2025-08-01')).toBe('25/26');
  });

  it('keeps January in the year that started the previous August', () => {
    expect(kitaYearLabel('2026-01-01')).toBe('25/26');
  });

  it('handles a century boundary without losing the leading digits', () => {
    expect(kitaYearLabel('2099-08-01')).toBe('99/00');
  });
});

describe('buildKitaYearCompareWindows', () => {
  it('returns nothing when no month has a bill', () => {
    expect(buildKitaYearCompareWindows([])).toEqual([]);
  });

  it('clamps a window to the months actually billed', () => {
    // Only Feb-Apr 2026 billed: the window must not ask for Aug 2025-Jul 2026.
    expect(buildKitaYearCompareWindows(['2026-02-01', '2026-03-01', '2026-04-01'])).toEqual([
      { kitaYear: '25/26', from: '2026-02-01', to: '2026-04-01' },
    ]);
  });

  it('cuts on the Kita-year boundary rather than on a 12-month block', () => {
    // 18 consecutive billed months from Feb 2026. The old 12-month-block
    // builder produced Feb 2026-Jan 2027, which straddles the Jul/Aug boundary
    // and left a Kita year matched to a window holding the next year's months.
    const months = [
      '2026-02-01',
      '2026-03-01',
      '2026-04-01',
      '2026-05-01',
      '2026-06-01',
      '2026-07-01',
      '2026-08-01',
      '2026-09-01',
      '2026-10-01',
      '2026-11-01',
      '2026-12-01',
      '2027-01-01',
      '2027-02-01',
      '2027-03-01',
      '2027-04-01',
      '2027-05-01',
      '2027-06-01',
      '2027-07-01',
    ];
    expect(buildKitaYearCompareWindows(months)).toEqual([
      { kitaYear: '25/26', from: '2026-02-01', to: '2026-07-01' },
      { kitaYear: '26/27', from: '2026-08-01', to: '2027-07-01' },
    ]);
  });

  it('never returns a window longer than one Kita year', () => {
    const months: string[] = [];
    for (let year = 2024; year <= 2027; year++) {
      for (let month = 1; month <= 12; month++) {
        months.push(`${year}-${String(month).padStart(2, '0')}-01`);
      }
    }
    for (const w of buildKitaYearCompareWindows(months)) {
      const from = new Date(`${w.from}T00:00:00`);
      const to = new Date(`${w.to}T00:00:00`);
      const span = (to.getFullYear() - from.getFullYear()) * 12 + (to.getMonth() - from.getMonth());
      expect(span).toBeLessThanOrEqual(11);
      expect(kitaYearLabel(w.from)).toBe(w.kitaYear);
      expect(kitaYearLabel(w.to)).toBe(w.kitaYear);
    }
  });

  it('tolerates unsorted, duplicated and empty input', () => {
    expect(
      buildKitaYearCompareWindows(['2026-04-01', '', '2026-02-01', '2026-04-01', '2026-03-01'])
    ).toEqual([{ kitaYear: '25/26', from: '2026-02-01', to: '2026-04-01' }]);
  });

  it('gives a gap in the middle of a Kita year one window spanning the gap', () => {
    // CompareRange skips months without a bill, so one window is cheaper than
    // two and returns the same comparisons.
    expect(buildKitaYearCompareWindows(['2025-09-01', '2026-03-01'])).toEqual([
      { kitaYear: '25/26', from: '2025-09-01', to: '2026-03-01' },
    ]);
  });

  it('orders windows chronologically regardless of input order', () => {
    const windows = buildKitaYearCompareWindows(['2027-01-01', '2025-09-01', '2026-09-01']);
    expect(windows.map((w) => w.kitaYear)).toEqual(['25/26', '26/27']);
  });
});
