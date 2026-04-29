import { FinancialSummaryChart } from '../financial-summary-chart';
import { renderWithProviders } from '@/test-utils';
import type { FinancialResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@nivo/bar', () => ({
  ResponsiveBar: () => <div data-testid="nivo-bar" />,
}));

function makeDataPoint(
  overrides: Partial<FinancialResponse['data_points'][0]>
): FinancialResponse['data_points'][0] {
  return {
    date: '2024-08-01',
    funding_income: 0,
    actual_funding: 0,
    actual_funding_correction: 0,
    actual_funding_regular: 0,
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

const emptyData: FinancialResponse = {
  data_points: [],
  warnings: [],
};

const sampleData: FinancialResponse = {
  data_points: [
    makeDataPoint({
      date: '2024-08-01',
      funding_income: 500000,
      budget_income: 100000,
      gross_salary: 300000,
      employer_costs: 50000,
      budget_expenses: 80000,
      total_income: 600000,
      total_expenses: 430000,
      balance: 170000,
      child_count: 20,
      staff_count: 5,
    }),
    makeDataPoint({
      date: '2024-09-01',
      funding_income: 510000,
      budget_income: 110000,
      gross_salary: 310000,
      employer_costs: 55000,
      budget_expenses: 85000,
      total_income: 620000,
      total_expenses: 450000,
      balance: -30000,
      child_count: 21,
      staff_count: 5,
    }),
  ],
  warnings: [],
};

describe('FinancialSummaryChart', () => {
  it('renders without crashing with empty data', () => {
    const { container } = renderWithProviders(<FinancialSummaryChart data={emptyData} />);
    expect(container).toBeTruthy();
  });

  it('renders the chart with data points', () => {
    const { container } = renderWithProviders(<FinancialSummaryChart data={sampleData} />);
    expect(container).toBeTruthy();
  });
});
