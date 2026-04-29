import { FinancialsChart } from '../financials-bar-chart';
import { renderWithProviders } from '@/test-utils';
import type { FinancialResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

// Mock Nivo's ResponsiveBar since it requires SVG measurements not available in JSDOM
jest.mock('@nivo/bar', () => ({
  ResponsiveBar: () => <div data-testid="nivo-bar" />,
}));

jest.mock('d3-shape', () => ({
  line: () => {
    const fn = () => '';
    fn.x = () => fn;
    fn.y = () => fn;
    fn.curve = () => fn;
    fn.defined = () => fn;
    return fn;
  },
  curveMonotoneX: {},
}));

function makeDataPoint(
  overrides: Partial<FinancialResponse['data_points'][0]>
): FinancialResponse['data_points'][0] {
  return {
    date: '2024-01-01',
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
      date: '2024-01-01',
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
      date: '2024-02-01',
      funding_income: 510000,
      budget_income: 110000,
      gross_salary: 310000,
      employer_costs: 55000,
      budget_expenses: 85000,
      total_income: 620000,
      total_expenses: 450000,
      balance: 170000,
      child_count: 21,
      staff_count: 5,
    }),
  ],
  warnings: [],
};

describe('FinancialsChart', () => {
  it('renders without crashing with empty data', () => {
    const { container } = renderWithProviders(<FinancialsChart data={emptyData} />);
    expect(container).toBeTruthy();
  });

  it('renders the chart with data points', () => {
    const { container } = renderWithProviders(<FinancialsChart data={sampleData} />);
    expect(container).toBeTruthy();
  });
});
