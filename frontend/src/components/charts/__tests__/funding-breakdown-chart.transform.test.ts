jest.mock('@nivo/pie', () => ({ ResponsivePie: () => null }));

import { buildFundingSlices, FUNDING_BREAKDOWN_COLORS } from '../funding-breakdown-chart';
import type {
  FinancialBudgetItemDetail,
  FinancialDataPoint,
  FinancialFundingDetail,
} from '@/lib/api/types';

const baseDp: FinancialDataPoint = {
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
};

function makeFunding(overrides: Partial<FinancialFundingDetail> = {}): FinancialFundingDetail {
  return {
    key: 'care_type',
    value: 'ganztag',
    label: 'Ganztag',
    amount_cents: 0,
    ...overrides,
  };
}

function makeBudget(overrides: Partial<FinancialBudgetItemDetail> = {}): FinancialBudgetItemDetail {
  return {
    name: 'Item',
    category: 'income',
    amount_cents: 0,
    unit_amount_cents: 0,
    per_child: false,
    ...overrides,
  };
}

describe('buildFundingSlices', () => {
  describe('empty / no-input branches', () => {
    it('returns [] when neither funding nor income budget items exist', () => {
      // Triggers the chart's "no data" early-return.
      expect(buildFundingSlices(baseDp)).toEqual([]);
    });

    it('returns [] when only zero-amount entries are present', () => {
      // amount_cents > 0 strict — equal-to-zero entries are skipped.
      const dp = {
        ...baseDp,
        funding_details: [makeFunding({ amount_cents: 0 })],
        budget_item_details: [makeBudget({ category: 'income', amount_cents: 0 })],
      };
      expect(buildFundingSlices(dp)).toEqual([]);
    });

    it('returns [] when funding_details / budget_item_details are undefined', () => {
      const dp = {
        ...baseDp,
        funding_details: undefined,
        budget_item_details: undefined,
      } as unknown as FinancialDataPoint;
      expect(buildFundingSlices(dp)).toEqual([]);
    });
  });

  describe('government funding slices', () => {
    it('emits one slice per non-zero funding detail with composite id', () => {
      // ID is `funding_${key}_${value}` so two entries with same `key`
      // but different `value` (e.g. care_type=halbtag vs ganztag) get
      // distinct ids and are not deduplicated.
      const dp = {
        ...baseDp,
        funding_details: [
          makeFunding({ key: 'care_type', value: 'ganztag', label: 'Ganztag', amount_cents: 1000 }),
          makeFunding({ key: 'care_type', value: 'halbtag', label: 'Halbtag', amount_cents: 500 }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices).toHaveLength(2);
      expect(slices.map((s) => s.id)).toEqual([
        'funding_care_type_ganztag',
        'funding_care_type_halbtag',
      ]);
    });

    it('uses the funding label as display label, falling back to empty', () => {
      const dp = {
        ...baseDp,
        funding_details: [
          makeFunding({ key: 'k1', value: 'v1', label: 'Visible', amount_cents: 100 }),
          makeFunding({
            key: 'k2',
            value: 'v2',
            label: undefined as unknown as string,
            amount_cents: 100,
          }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices[0]?.label).toBe('Visible');
      expect(slices[1]?.label).toBe('');
    });

    it('skips zero-amount funding entries but keeps non-zero ones in input order', () => {
      const dp = {
        ...baseDp,
        funding_details: [
          makeFunding({ key: 'a', value: 'a', amount_cents: 100 }),
          makeFunding({ key: 'b', value: 'b', amount_cents: 0 }),
          makeFunding({ key: 'c', value: 'c', amount_cents: 50 }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices.map((s) => s.id)).toEqual(['funding_a_a', 'funding_c_c']);
    });

    it('handles undefined amount_cents (older payloads) as 0 → skip', () => {
      const dp = {
        ...baseDp,
        funding_details: [
          {
            key: 'a',
            value: 'a',
            label: 'X',
            // amount_cents intentionally missing
          } as unknown as FinancialFundingDetail,
        ],
      };
      expect(buildFundingSlices(dp)).toEqual([]);
    });
  });

  describe('budget income slices (always appended after funding)', () => {
    it("filters out items whose category is not 'income'", () => {
      // Expense items must NOT appear in the funding-breakdown chart.
      // They go into the expense breakdown instead. Same DTO, different
      // chart — the filter is the only thing keeping them apart.
      const dp = {
        ...baseDp,
        budget_item_details: [
          makeBudget({ name: 'Parent fees', category: 'income', amount_cents: 5000 }),
          makeBudget({ name: 'Toys', category: 'expense', amount_cents: 9000 }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices).toHaveLength(1);
      expect(slices[0]?.label).toBe('Parent fees');
    });

    it('skips zero-amount income items', () => {
      const dp = {
        ...baseDp,
        budget_item_details: [
          makeBudget({ name: 'Z', category: 'income', amount_cents: 0 }),
          makeBudget({ name: 'NZ', category: 'income', amount_cents: 1 }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices.map((s) => s.label)).toEqual(['NZ']);
    });

    it('uses the budget name as id (prefixed with budget_) and as label', () => {
      const dp = {
        ...baseDp,
        budget_item_details: [
          makeBudget({ name: 'Elternbeiträge', category: 'income', amount_cents: 12345 }),
        ],
      };
      const slices = buildFundingSlices(dp);
      expect(slices[0]).toMatchObject({
        id: 'budget_Elternbeiträge',
        label: 'Elternbeiträge',
        value: 123.45,
      });
    });

    it('appends budget slices AFTER funding slices (preserves visual order)', () => {
      const dp = {
        ...baseDp,
        funding_details: [makeFunding({ key: 'a', value: 'a', amount_cents: 100 })],
        budget_item_details: [makeBudget({ name: 'B', category: 'income', amount_cents: 50 })],
      };
      const slices = buildFundingSlices(dp);
      expect(slices.map((s) => s.id)).toEqual(['funding_a_a', 'budget_B']);
    });
  });

  describe('color cycling', () => {
    it('cycles colors across the combined funding + budget arrays', () => {
      // Important: the colorIdx counter is shared across both loops.
      // 3 funding entries + 4 budget entries → 7 slices, palette has 6 →
      // last slice wraps back to colors[0].
      const dp = {
        ...baseDp,
        funding_details: Array.from({ length: 3 }, (_, i) =>
          makeFunding({ key: `f${i}`, value: `v${i}`, amount_cents: 100 })
        ),
        budget_item_details: Array.from({ length: 4 }, (_, i) =>
          makeBudget({ name: `b${i}`, category: 'income', amount_cents: 100 })
        ),
      };
      const slices = buildFundingSlices(dp);
      expect(slices).toHaveLength(7);
      slices.forEach((s, i) => {
        expect(s.color).toBe(FUNDING_BREAKDOWN_COLORS[i % FUNDING_BREAKDOWN_COLORS.length]);
      });
      expect(slices[0]?.color).toBe(slices[6]?.color);
    });

    it('respects an injected colors array (test ergonomics)', () => {
      const dp = {
        ...baseDp,
        funding_details: [makeFunding({ amount_cents: 100 })],
      };
      const slices = buildFundingSlices(dp, ['#deadbeef']);
      expect(slices[0]?.color).toBe('#deadbeef');
    });
  });

  describe('value precision', () => {
    it('returns euros (cents/100), not cents', () => {
      const dp = {
        ...baseDp,
        funding_details: [makeFunding({ amount_cents: 166847 })],
      };
      expect(buildFundingSlices(dp)[0]?.value).toBe(1668.47);
    });
  });
});
