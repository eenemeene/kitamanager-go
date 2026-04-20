import { screen } from '@testing-library/react';
import { BudgetTable } from '../budget-table';
import { renderWithProviders } from '@/test-utils';
import type { FinancialResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

function makeDataPoint(
  date: string,
  overrides: Partial<FinancialResponse['data_points'][0]> = {}
): FinancialResponse['data_points'][0] {
  return {
    date,
    funding_income: 500000,
    gross_salary: 200000,
    employer_costs: 50000,
    budget_income: 10000,
    budget_expenses: 5000,
    total_income: 510000,
    total_expenses: 255000,
    balance: 255000,
    child_count: 10,
    staff_count: 5,
    ...overrides,
  };
}

const threeMonthData: FinancialResponse = {
  data_points: [
    makeDataPoint('2026-01-01'),
    makeDataPoint('2026-02-01'),
    makeDataPoint('2026-03-01'),
  ],
};

describe('BudgetTable', () => {
  it('renders month rows plus total row', () => {
    renderWithProviders(<BudgetTable data={threeMonthData} />);

    // 3 month rows + 1 total row
    expect(screen.getByText('annualTotal')).toBeInTheDocument();
    // Check that month headers appear (3 data rows + 1 total row = 4 body rows)
    const rows = document.querySelectorAll('tbody tr');
    expect(rows.length).toBe(4);
  });

  it('renders column headers', () => {
    renderWithProviders(<BudgetTable data={threeMonthData} />);

    expect(screen.getByText('totalIncome')).toBeInTheDocument();
    expect(screen.getByText('totalExpenses')).toBeInTheDocument();
    expect(screen.getByText('balance')).toBeInTheDocument();
    expect(screen.getByText('fundingIncomeSub')).toBeInTheDocument();
    expect(screen.getByText('salaries')).toBeInTheDocument();
    expect(screen.getByText('incomeTotal')).toBeInTheDocument();
    expect(screen.getByText('expenseTotal')).toBeInTheDocument();
  });

  it('formats currency values', () => {
    const data: FinancialResponse = {
      data_points: [makeDataPoint('2026-01-01', { total_income: 166847 })],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // formatCurrency(166847) in de locale -> "1.668,47 €"
    // Appears in both the month row and the total row
    const matches = screen.getAllByText(/1\.668,47/);
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it('shows annual totals that sum all months', () => {
    const data: FinancialResponse = {
      data_points: [
        makeDataPoint('2026-01-01', { funding_income: 100000 }),
        makeDataPoint('2026-02-01', { funding_income: 200000 }),
      ],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // Total funding = 100000 + 200000 = 300000 cents = 3.000,00 €
    expect(screen.getByText(/3\.000,00/)).toBeInTheDocument();
  });

  it('handles empty data gracefully', () => {
    const data: FinancialResponse = { data_points: [] };

    renderWithProviders(<BudgetTable data={data} />);

    expect(screen.getByText('chartError')).toBeInTheDocument();
  });

  it('renders budget item detail columns', () => {
    const data: FinancialResponse = {
      data_points: [
        makeDataPoint('2026-01-01', {
          budget_item_details: [
            { name: 'Elternbeiträge', category: 'income', amount_cents: 9000 },
            { name: 'Miete', category: 'expense', amount_cents: 12000 },
          ],
        }),
      ],
    };

    renderWithProviders(<BudgetTable data={data} />);

    expect(screen.getByText('Elternbeiträge')).toBeInTheDocument();
    expect(screen.getByText('Miete')).toBeInTheDocument();
  });

  it('applies green class to positive balance and red to negative', () => {
    const data: FinancialResponse = {
      data_points: [
        makeDataPoint('2026-01-01', { balance: 100000 }),
        makeDataPoint('2026-02-01', { balance: -50000 }),
      ],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // Positive: 1.000,00 €
    const positiveCell = screen.getByText(/1\.000,00/).closest('td');
    expect(positiveCell?.className).toMatch(/text-success/);

    // Negative: -500,00 €
    const negativeCell = screen.getByText(/-500,00/).closest('td');
    expect(negativeCell?.className).toMatch(/text-destructive/);
  });

  it('shows dash for zero values', () => {
    const data: FinancialResponse = {
      data_points: [
        makeDataPoint('2026-01-01', {
          funding_income: 0,
          gross_salary: 0,
          employer_costs: 0,
          budget_income: 0,
          budget_expenses: 0,
          total_income: 0,
          total_expenses: 0,
          balance: 0,
        }),
      ],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // All monetary cells should show em-dash for zero values
    const cells = document.querySelectorAll('td.tabular-nums');
    const dashCells = Array.from(cells).filter((cell) => cell.textContent === '\u2013');
    expect(dashCells.length).toBeGreaterThan(0);
  });

  it('handles missing budget_item_details', () => {
    const data: FinancialResponse = {
      data_points: [makeDataPoint('2026-01-01', { budget_item_details: undefined })],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // Should still render without crashing
    expect(screen.getByText('annualTotal')).toBeInTheDocument();
  });

  it('handles budget items appearing in some months but not others', () => {
    const data: FinancialResponse = {
      data_points: [
        makeDataPoint('2026-01-01', {
          budget_item_details: [{ name: 'Spenden', category: 'income', amount_cents: 5000 }],
        }),
        makeDataPoint('2026-02-01', {
          budget_item_details: [],
        }),
      ],
    };

    renderWithProviders(<BudgetTable data={data} />);

    // Spenden column header should exist
    expect(screen.getByText('Spenden')).toBeInTheDocument();
    // Should render without error
    expect(screen.getByText('annualTotal')).toBeInTheDocument();
  });
});
