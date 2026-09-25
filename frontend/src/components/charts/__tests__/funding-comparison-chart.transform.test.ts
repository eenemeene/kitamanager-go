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

/**
 * A billed month whose attributed figures differ from its arrival ones --
 * the shape a bill produces when it carries corrections for earlier months.
 */
function billedAttributed(
  date: string,
  calculated: number,
  arrival: { regular: number; correction: number },
  attributed: { regular: number; correction: number }
): DataPoint {
  return dp(date, {
    funding_income: calculated,
    actual_funding: arrival.regular + arrival.correction,
    actual_funding_regular: arrival.regular,
    actual_funding_correction: arrival.correction,
    actual_funding_regular_attributed: attributed.regular,
    actual_funding_correction_attributed: attributed.correction,
  });
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

  // ------------------------------------------------------------------
  // Attribution: which month a correction counts against
  // ------------------------------------------------------------------

  // The defect. A bill for April carries corrections for January, February
  // and March; keyed by arrival, all of it counted as April money, so the
  // year row compared corrections for one Kita year against a calculated
  // total for another.
  it('counts a correction against the month it corrects, not the month it arrived', () => {
    const rows = buildKitaYearSummary([
      // March: corrected by the April bill, 102 attributed back to it.
      billedAttributed(
        '2026-03-01',
        1000,
        { regular: 1000, correction: 0 },
        { regular: 1000, correction: 102 }
      ),
      // April: the bill arrived here carrying that correction, but only its
      // own regular row is about April.
      billedAttributed(
        '2026-04-01',
        1000,
        { regular: 1000, correction: 102 },
        { regular: 1000, correction: 0 }
      ),
    ]);
    const row = rows[0]!;
    expect(row.regular).toBe(2000);
    expect(row.correction).toBe(102);
    // Per-month: the correction shows on March, not April.
    expect(row.months.map((m) => m.correction)).toEqual([102, 0]);
  });

  // Both keyings describe the same money, so whichever the row is built from,
  // the total may not change -- only which month it lands on.
  it('moves a correction between months without changing the total', () => {
    const rows = buildKitaYearSummary([
      billedAttributed(
        '2026-03-01',
        1000,
        { regular: 1000, correction: 0 },
        { regular: 1000, correction: 102 }
      ),
      billedAttributed(
        '2026-04-01',
        1000,
        { regular: 1000, correction: 102 },
        { regular: 1000, correction: 0 }
      ),
    ]);
    const row = rows[0]!;
    expect(row.regular + row.correction).toBe(2102);
    expect(row.difference).toBe(row.regular + row.correction - row.calculatedWithBill);
  });

  // Bills imported before the billing month was persisted report no attributed
  // figures at all. Those rows must read exactly as they did before.
  it('falls back to the arrival figures when no attributed ones are present', () => {
    const rows = buildKitaYearSummary([billed('2025-09-01', 1000, 900, 50)]);
    const row = rows[0]!;
    expect(row.regular).toBe(900);
    expect(row.correction).toBe(50);
    expect(row.orphanCorrection).toBe(0);
    expect(row.difference).toBe(-50);
  });

  // A correction can be attributed to a month no bill was ever imported for.
  // Folding it into `correction` would set 102 against a calculated total that
  // excludes that month entirely, reading as a month-sized deficit; dropping it
  // would lose money the Senate actually paid. It is held out and reported.
  it('holds a correction for an unbilled month aside rather than into the difference', () => {
    const rows = buildKitaYearSummary([
      // No bill of its own, but the April bill corrected it.
      dp('2025-08-01', { funding_income: 50000, actual_funding_correction_attributed: 102 }),
      billed('2025-09-01', 1000, 1000),
    ]);
    const row = rows[0]!;
    expect(row.orphanCorrection).toBe(102);
    expect(row.correction).toBe(0);
    // The unbilled month contributes to neither side of the difference.
    expect(row.calculatedWithBill).toBe(1000);
    expect(row.difference).toBe(0);
    expect(row.actualMonths).toBe(1);
    expect(row.totalMonths).toBe(2);
  });

  // The month row for an unbilled month stays blank: an attributed correction
  // does not make the month evaluable, and printing it beside an empty
  // Calculated cell would invite exactly the subtraction that is wrong.
  it("leaves an unbilled month's cells empty even when it carries an attributed correction", () => {
    const rows = buildKitaYearSummary([
      dp('2025-08-01', { funding_income: 50000, actual_funding_correction_attributed: 102 }),
    ]);
    const august = rows[0]!.months[0]!;
    expect(august.regular).toBeNull();
    expect(august.correction).toBeNull();
    expect(august.difference).toBeNull();
  });
});
