// The component file imports `@nivo/bar`, which ships ESM that jest's next/jest
// config doesn't transform. Stubbing it so importing the transform doesn't pull
// d3 through. The transform uses no Nivo APIs, so the stub is invisible here.
jest.mock('@nivo/bar', () => ({ ResponsiveBar: () => null }));

import { buildKitaYearSummary } from '../funding-comparison-chart';
import type { FinancialResponse } from '@/lib/api/types';

type DataPoint = FinancialResponse['data_points'][0];

function dp(date: string, overrides: Partial<DataPoint> = {}): DataPoint {
  return {
    date,
    funding_income: 0,
    budget_income: 0,
    gross_salary: 0,
    employer_costs: 0,
    budget_expenses: 0,
    budget_item_details: [],
    funding_details: [],
    salary_details: [],
    total_income: 0,
    total_expenses: 0,
    balance: 0,
    child_count: 0,
    staff_count: 0,
    ...overrides,
  };
}

/** A month with a bill: actual_funding present is what marks it as billed. */
function billed(date: string, calculated: number, regular: number, correction = 0): DataPoint {
  return dp(date, {
    funding_income: calculated,
    actual_funding: regular + correction,
    actual_funding_regular: regular,
    actual_funding_correction: correction,
  });
}

/** A month with no bill uploaded: only the calculated side exists. */
function unbilled(date: string, calculated: number): DataPoint {
  return dp(date, { funding_income: calculated });
}

describe('buildKitaYearSummary', () => {
  it('groups months into Kita years running August to July', () => {
    const rows = buildKitaYearSummary([
      unbilled('2025-07-01', 100),
      unbilled('2025-08-01', 100),
      unbilled('2026-07-01', 100),
      unbilled('2026-08-01', 100),
    ]);
    expect(rows.map((r) => r.label)).toEqual(['24/25', '25/26', '26/27']);
    expect(rows.map((r) => r.totalMonths)).toEqual([1, 2, 1]);
  });

  // The defect this file exists for. With six of twelve months billed, the
  // table used to print calculatedTotal next to a difference built from
  // calculatedWithBill, so subtracting the cells on screen gave a number
  // hundreds of thousands of euros away from the one beside them.
  it('exposes the calculated figure the difference is actually built from', () => {
    const rows = buildKitaYearSummary([
      billed('2025-08-01', 1000, 1100),
      billed('2025-09-01', 1000, 1100),
      unbilled('2025-10-01', 5000),
      unbilled('2025-11-01', 5000),
    ]);
    const year = rows[0]!;

    expect(year.calculatedWithBill).toBe(2000);
    expect(year.calculatedTotal).toBe(12000);
    expect(year.complete).toBe(false);
    // The row must be reproducible from the figures it is built from.
    expect(year.difference).toBe(year.regular + year.correction - year.calculatedWithBill);
    // And emphatically not from the full-range total, which is the old bug.
    expect(year.difference).not.toBe(year.regular + year.correction - year.calculatedTotal);
  });

  it('collapses the two calculated figures when every month is billed', () => {
    const rows = buildKitaYearSummary([
      billed('2025-08-01', 1000, 1100),
      billed('2025-09-01', 1000, 1100),
    ]);
    const year = rows[0]!;
    expect(year.complete).toBe(true);
    expect(year.calculatedWithBill).toBe(year.calculatedTotal);
    expect(year.difference).toBe(200);
  });

  it('reports a year with no bills as having none, and totals the full range', () => {
    const rows = buildKitaYearSummary([unbilled('2025-08-01', 1000), unbilled('2025-09-01', 1000)]);
    const year = rows[0]!;
    expect(year.hasBills).toBe(false);
    expect(year.actualMonths).toBe(0);
    expect(year.calculatedTotal).toBe(2000);
    expect(year.calculatedWithBill).toBe(0);
  });

  // dcdcd2d8 decided this split deliberately: the year row answers "what did we
  // net after retroactive adjustments", the month rows answer "is this month
  // billed right". A correction pays for a PRIOR month, so it belongs in the
  // first answer and not the second. These assertions exist so the split stays
  // a decision rather than drifting.
  describe('corrections', () => {
    const rows = buildKitaYearSummary([
      billed('2025-08-01', 1000, 1000, 500),
      billed('2025-09-01', 1000, 900),
    ]);
    const year = rows[0]!;

    it('counts corrections in the year difference', () => {
      expect(year.regular).toBe(1900);
      expect(year.correction).toBe(500);
      expect(year.difference).toBe(1900 + 500 - 2000);
    });

    it('leaves corrections out of a month difference', () => {
      const august = year.months.find((m) => m.date === '2025-08-01')!;
      expect(august.correction).toBe(500);
      expect(august.difference).toBe(0); // regular 1000 - calculated 1000
    });

    it('keeps the correction visible on the month row even though it is excluded', () => {
      expect(year.months.map((m) => m.correction)).toEqual([500, 0]);
    });
  });

  it('leaves a month difference null when that month has no bill', () => {
    const rows = buildKitaYearSummary([
      billed('2025-08-01', 1000, 1100),
      unbilled('2025-09-01', 1000),
    ]);
    const [august, september] = rows[0]!.months;
    expect(august!.difference).toBe(100);
    expect(september!.difference).toBeNull();
    expect(september!.regular).toBeNull();
  });

  it('treats a missing funding_income as zero rather than dropping the month', () => {
    const rows = buildKitaYearSummary([dp('2025-08-01')]);
    expect(rows[0]!.totalMonths).toBe(1);
    expect(rows[0]!.calculatedTotal).toBe(0);
  });

  it('carries per-month comparison counts through when compareData is supplied', () => {
    const compareData = new Map([
      [
        '2025-08-01',
        {
          bill_only_count: 2,
          bill_only_amount: 300,
          calc_only_count: 1,
          calc_only_amount: 150,
        },
      ],
    ]) as never;
    const rows = buildKitaYearSummary([billed('2025-08-01', 1000, 1100)], compareData);
    const august = rows[0]!.months[0]!;
    expect(august.billOnlyCount).toBe(2);
    expect(august.calcOnlyAmount).toBe(150);
  });
});
