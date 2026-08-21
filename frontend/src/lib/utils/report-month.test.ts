import { parseReportMonth, formatReportMonthLong, formatKitaYearLabel } from './report-month';

describe('parseReportMonth', () => {
  it('parses a valid YYYY-MM input', () => {
    const m = parseReportMonth('2026-04');
    expect(m.month).toBe('2026-04');
    expect(m.asOf).toBe('2026-04-01');
    expect(m.year).toBe(2026);
    expect(m.monthNumber).toBe(4);
  });

  it('falls back to the current month when input is null', () => {
    const m = parseReportMonth(null);
    const now = new Date();
    expect(m.year).toBe(now.getFullYear());
    expect(m.monthNumber).toBe(now.getMonth() + 1);
  });

  it('falls back to the current month when input is undefined', () => {
    const m = parseReportMonth(undefined);
    const now = new Date();
    expect(m.year).toBe(now.getFullYear());
    expect(m.monthNumber).toBe(now.getMonth() + 1);
  });

  it('falls back when input has wrong separator', () => {
    const m = parseReportMonth('2026/04');
    const now = new Date();
    expect(m.year).toBe(now.getFullYear());
    expect(m.monthNumber).toBe(now.getMonth() + 1);
  });

  it('falls back when month is out of 1-12 range', () => {
    const m = parseReportMonth('2026-13');
    const now = new Date();
    expect(m.year).toBe(now.getFullYear());
  });

  it('falls back when year is out of 2000-2100 range', () => {
    const m = parseReportMonth('1999-04');
    const now = new Date();
    expect(m.year).toBe(now.getFullYear());
  });

  it('zero-pads single-digit months in output', () => {
    const m = parseReportMonth('2026-03');
    expect(m.month).toBe('2026-03');
    expect(m.asOf).toBe('2026-03-01');
  });

  describe('Kita year window (Aug → Jul)', () => {
    it('places report months Aug-Dec into the Kita year starting that August', () => {
      const m = parseReportMonth('2026-09');
      expect(m.kitaYearStartYear).toBe(2026);
      expect(m.kitaYearFrom).toBe('2026-08-01');
      expect(m.kitaYearTo).toBe('2027-07-01');
    });

    it('places report months Jan-Jul into the Kita year starting the previous August', () => {
      const m = parseReportMonth('2026-04');
      expect(m.kitaYearStartYear).toBe(2025);
      expect(m.kitaYearFrom).toBe('2025-08-01');
      expect(m.kitaYearTo).toBe('2026-07-01');
    });

    it('handles July (still belongs to the previous Kita year)', () => {
      const m = parseReportMonth('2026-07');
      expect(m.kitaYearStartYear).toBe(2025);
      expect(m.kitaYearFrom).toBe('2025-08-01');
      expect(m.kitaYearTo).toBe('2026-07-01');
    });

    it('handles August (start of a new Kita year)', () => {
      const m = parseReportMonth('2026-08');
      expect(m.kitaYearStartYear).toBe(2026);
      expect(m.kitaYearFrom).toBe('2026-08-01');
      expect(m.kitaYearTo).toBe('2027-07-01');
    });
  });

  describe('Trend window (prev + current + next Kita year)', () => {
    it('spans 3 Kita years for a mid-Kita-year report month', () => {
      const m = parseReportMonth('2026-04'); // Kita year: 2025-08 → 2026-07
      expect(m.trendFrom).toBe('2024-08-01');
      expect(m.trendTo).toBe('2027-07-01');
    });

    it('spans 3 Kita years for an Aug report month', () => {
      const m = parseReportMonth('2026-08'); // Kita year: 2026-08 → 2027-07
      expect(m.trendFrom).toBe('2025-08-01');
      expect(m.trendTo).toBe('2028-07-01');
    });

    it('spans 36 months from trendFrom to trendTo (inclusive of both endpoints)', () => {
      const m = parseReportMonth('2026-04');
      const start = new Date(m.trendFrom);
      const end = new Date(m.trendTo);
      const months =
        (end.getUTCFullYear() - start.getUTCFullYear()) * 12 +
        (end.getUTCMonth() - start.getUTCMonth()) +
        1;
      expect(months).toBe(36);
    });
  });
});

describe('formatReportMonthLong', () => {
  it('formats with month name + year in English', () => {
    const m = parseReportMonth('2026-04');
    expect(formatReportMonthLong(m, 'en')).toBe('April 2026');
  });

  it('formats December correctly', () => {
    const m = parseReportMonth('2025-12');
    expect(formatReportMonthLong(m, 'en')).toBe('December 2025');
  });
});

describe('formatKitaYearLabel', () => {
  it('formats the Kita-year span for a mid-Kita-year report month', () => {
    const m = parseReportMonth('2026-04');
    expect(formatKitaYearLabel(m, 'en')).toBe('Aug 2025 – Jul 2026');
  });

  it('formats the Kita-year span for an Aug report month', () => {
    const m = parseReportMonth('2026-08');
    expect(formatKitaYearLabel(m, 'en')).toBe('Aug 2026 – Jul 2027');
  });

  it('uses the requested locale for month abbreviations', () => {
    const m = parseReportMonth('2026-04');
    // German: Aug. 2025 – Juli 2026 (month abbreviations vary)
    const label = formatKitaYearLabel(m, 'de');
    expect(label).toContain('2025');
    expect(label).toContain('2026');
    expect(label).toContain('–');
  });
});
