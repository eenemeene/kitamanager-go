// The component file imports `@nivo/pie`, which ships ESM that jest's
// next/jest config doesn't transform. Stubbing the module so just
// importing the transform doesn't pull d3 through. The transform itself
// uses no Nivo APIs, so this stub is invisible to the tests below.
jest.mock('@nivo/pie', () => ({ ResponsivePie: () => null }));

// eslint-disable-next-line import/first
import { buildExpenseSlices, EXPENSE_BREAKDOWN_COLORS } from '../expense-breakdown-chart';
// eslint-disable-next-line import/first
import type {
  FinancialBudgetItemDetail,
  FinancialDataPoint,
  FinancialSalaryDetail,
} from '@/lib/api/types';

// Pure transform under test. No render. No Nivo. We pin every branch
// in `buildExpenseSlices` and assert on the slice array shape, ordering,
// values (cents → euros), color cycling, and the salaryDetail attachment
// the tooltip relies on.

const t = (key: string) => key;

function makeDp(overrides: Partial<FinancialDataPoint> = {}): FinancialDataPoint {
  return {
    date: '2025-01-01',
    total_income: 0,
    total_expenses: 0,
    balance: 0,
    funding_income: 0,
    actual_funding: 0,
    actual_funding_regular: 0,
    actual_funding_correction: 0,
    gross_salary: 0,
    employer_costs: 0,
    budget_income: 0,
    budget_expenses: 0,
    child_count: 0,
    staff_count: 0,
    salary_details: [],
    funding_details: [],
    budget_item_details: [],
    ...overrides,
  };
}

function makeSalary(overrides: Partial<FinancialSalaryDetail> = {}): FinancialSalaryDetail {
  return {
    staff_category: 'qualified',
    gross_salary: 0,
    employer_costs: 0,
    ...overrides,
  };
}

function makeBudget(overrides: Partial<FinancialBudgetItemDetail> = {}): FinancialBudgetItemDetail {
  return {
    name: 'Item',
    category: 'expense',
    amount_cents: 0,
    unit_amount_cents: 0,
    per_child: false,
    ...overrides,
  };
}

describe('buildExpenseSlices', () => {
  describe('empty / no-input branches', () => {
    it('returns [] when data has no salary detail, no aggregate salary, and no budget details', () => {
      // Triggers the "render error message" branch in the component.
      const slices = buildExpenseSlices(makeDp(), t);
      expect(slices).toEqual([]);
    });

    it('returns [] when salary_details is undefined and aggregates are zero/undefined', () => {
      // salary_details may be `undefined` from the spec (older payloads).
      const dp = makeDp({ salary_details: undefined, gross_salary: 0, employer_costs: 0 });
      expect(buildExpenseSlices(dp, t)).toEqual([]);
    });

    it('returns [] when only zero-valued salary categories are present', () => {
      // Per-category branch should skip 0-valued categories, not emit them.
      const dp = makeDp({
        salary_details: [
          makeSalary({ staff_category: 'qualified', gross_salary: 0, employer_costs: 0 }),
          makeSalary({ staff_category: 'supplementary', gross_salary: 0, employer_costs: 0 }),
        ],
      });
      expect(buildExpenseSlices(dp, t)).toEqual([]);
    });
  });

  describe('per-category salary slices (preferred branch)', () => {
    it('emits one slice per non-zero category combining gross + employer', () => {
      const dp = makeDp({
        salary_details: [
          makeSalary({
            staff_category: 'qualified',
            gross_salary: 300_000,
            employer_costs: 66_000,
          }),
          makeSalary({
            staff_category: 'supplementary',
            gross_salary: 200_000,
            employer_costs: 44_000,
          }),
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices).toHaveLength(2);
      expect(slices[0]).toMatchObject({
        id: 'salary_qualified',
        label: 'employees.staffCategory.qualified',
        value: 3660, // (300_000 + 66_000) / 100 = 3660 €
      });
      expect(slices[1]).toMatchObject({
        id: 'salary_supplementary',
        value: 2440, // (200_000 + 44_000) / 100 = 2440 €
      });
    });

    it('attaches the salaryDetail object so the tooltip can show gross/employer split', () => {
      const sd = makeSalary({
        staff_category: 'qualified',
        gross_salary: 300_000,
        employer_costs: 66_000,
      });
      const slices = buildExpenseSlices(makeDp({ salary_details: [sd] }), t);
      expect(slices[0]?.salaryDetail).toBe(sd);
    });

    it('skips zero-total categories but keeps non-zero ones in input order', () => {
      const dp = makeDp({
        salary_details: [
          makeSalary({
            staff_category: 'qualified',
            gross_salary: 100_000,
            employer_costs: 22_000,
          }),
          makeSalary({ staff_category: 'supplementary', gross_salary: 0, employer_costs: 0 }),
          makeSalary({
            staff_category: 'non_pedagogical',
            gross_salary: 50_000,
            employer_costs: 0,
          }),
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices.map((s) => s.id)).toEqual(['salary_qualified', 'salary_non_pedagogical']);
    });

    it('treats undefined gross/employer (older payloads) as 0 rather than NaN', () => {
      // Important: the optional `?? 0` chain in the transform must not
      // produce NaN values that propagate into the chart and crash Nivo.
      const dp = makeDp({
        salary_details: [
          {
            staff_category: 'qualified',
            // intentionally omit gross/employer — strict types would
            // forbid this, but defensive runtime behaviour matters
            // since the API can drop them on older versions.
          } as unknown as FinancialSalaryDetail,
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      // total = 0 → category skipped
      expect(slices).toEqual([]);
    });

    it('does NOT fall through to aggregate salary when salary_details is non-empty (even if all zero)', () => {
      // Branch behaviour: the fallback only fires when salary_details
      // is missing/empty. An array of zero-valued entries still suppresses
      // aggregate slices to avoid double-counting.
      const dp = makeDp({
        salary_details: [
          makeSalary({ staff_category: 'qualified', gross_salary: 0, employer_costs: 0 }),
        ],
        gross_salary: 100_000,
        employer_costs: 22_000,
      });
      expect(buildExpenseSlices(dp, t)).toEqual([]);
    });
  });

  describe('aggregate-salary fallback branch', () => {
    it('emits gross_salary slice when set and salary_details is empty', () => {
      const dp = makeDp({ salary_details: [], gross_salary: 350_000 });
      const slices = buildExpenseSlices(dp, t);
      expect(slices).toHaveLength(1);
      expect(slices[0]).toMatchObject({
        id: 'gross_salary',
        label: 'statistics.grossSalary',
        value: 3500,
      });
    });

    it('emits employer_costs slice when set and salary_details is empty', () => {
      const dp = makeDp({ salary_details: [], employer_costs: 77_000 });
      const slices = buildExpenseSlices(dp, t);
      expect(slices).toHaveLength(1);
      expect(slices[0]?.id).toBe('employer_costs');
      expect(slices[0]?.value).toBe(770);
    });

    it('emits both gross + employer in stable order when both > 0', () => {
      const dp = makeDp({
        salary_details: [],
        gross_salary: 350_000,
        employer_costs: 77_000,
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices.map((s) => s.id)).toEqual(['gross_salary', 'employer_costs']);
    });

    it('skips negative or zero aggregate values', () => {
      // Edge case: a malformed/refunded payload could send negative totals.
      // The transform compares > 0 strictly — negatives are skipped.
      const dp = makeDp({
        salary_details: [],
        gross_salary: -100,
        employer_costs: 0,
      });
      expect(buildExpenseSlices(dp, t)).toEqual([]);
    });
  });

  describe('budget_item_details (always appended after salary)', () => {
    it("filters out items whose category is not 'expense'", () => {
      // Income items must NOT appear in the expense breakdown chart.
      const dp = makeDp({
        salary_details: [],
        gross_salary: 100_000,
        budget_item_details: [
          makeBudget({ name: 'Parent fees', category: 'income', amount_cents: 50_000 }),
          makeBudget({ name: 'Materials', category: 'expense', amount_cents: 30_000 }),
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      const budgetSlices = slices.filter((s) => s.id.startsWith('budget_'));
      expect(budgetSlices).toHaveLength(1);
      expect(budgetSlices[0]?.label).toBe('Materials');
    });

    it('filters out zero-amount expense items', () => {
      // amount_cents > 0 strict — a 0-amount expense item shouldn't
      // produce a slice (it would render as a 0%-arc and just be noise).
      const dp = makeDp({
        budget_item_details: [
          makeBudget({ name: 'A', category: 'expense', amount_cents: 0 }),
          makeBudget({ name: 'B', category: 'expense', amount_cents: 1 }),
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices.map((s) => s.label)).toEqual(['B']);
    });

    it('uses the budget item name as label and prefixes id with budget_', () => {
      const dp = makeDp({
        budget_item_details: [
          makeBudget({ name: 'Spielzeug', category: 'expense', amount_cents: 12_345 }),
        ],
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices[0]).toMatchObject({
        id: 'budget_Spielzeug',
        label: 'Spielzeug',
        value: 123.45,
      });
    });

    it('handles missing/undefined budget_item_details (older payloads)', () => {
      const dp = makeDp({
        salary_details: [],
        gross_salary: 100_000,
        budget_item_details: undefined,
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices).toHaveLength(1);
      expect(slices[0]?.id).toBe('gross_salary');
    });

    it('appends budget slices AFTER salary slices (preserves visual order)', () => {
      // Order matters: salary slices sit first in the legend / arc cycle.
      // A future refactor mustn't reorder them.
      const dp = makeDp({
        salary_details: [
          makeSalary({ staff_category: 'qualified', gross_salary: 100, employer_costs: 22 }),
        ],
        budget_item_details: [makeBudget({ name: 'X', category: 'expense', amount_cents: 50 })],
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices.map((s) => s.id)).toEqual(['salary_qualified', 'budget_X']);
    });
  });

  describe('color cycling', () => {
    it('assigns colors in palette order and cycles past the palette length', () => {
      // EXPENSE_BREAKDOWN_COLORS has 6 entries; with 8 slices the cycle wraps.
      const dp = makeDp({
        salary_details: Array.from({ length: 8 }, (_, i) =>
          makeSalary({
            staff_category: `cat_${i}` as FinancialSalaryDetail['staff_category'],
            gross_salary: 100,
            employer_costs: 0,
          })
        ),
      });
      const slices = buildExpenseSlices(dp, t);
      expect(slices).toHaveLength(8);
      slices.forEach((s, i) => {
        expect(s.color).toBe(EXPENSE_BREAKDOWN_COLORS[i % EXPENSE_BREAKDOWN_COLORS.length]);
      });
      // First and 7th (index 6) wrap back to color[0]
      expect(slices[0]?.color).toBe(slices[6]?.color);
    });

    it('uses an injected colors array when provided (test ergonomics)', () => {
      const dp = makeDp({
        salary_details: [
          makeSalary({ staff_category: 'qualified', gross_salary: 100, employer_costs: 0 }),
        ],
      });
      const slices = buildExpenseSlices(dp, t, ['#000000']);
      expect(slices[0]?.color).toBe('#000000');
    });
  });

  describe('value precision', () => {
    it('returns euros (cents/100) — not cents', () => {
      // Critical: Nivo arc labels and tooltips both expect euros at this
      // layer. A regression that left values as cents would render
      // pie tooltips as "300000,00 €" instead of "3000,00 €".
      const dp = makeDp({
        salary_details: [],
        gross_salary: 300_000,
      });
      expect(buildExpenseSlices(dp, t)[0]?.value).toBe(3000);
    });

    it('preserves fractional cents (e.g. 12345 cents → 123.45 €)', () => {
      const dp = makeDp({
        budget_item_details: [makeBudget({ name: 'X', category: 'expense', amount_cents: 12_345 })],
      });
      expect(buildExpenseSlices(dp, t)[0]?.value).toBe(123.45);
    });
  });
});
