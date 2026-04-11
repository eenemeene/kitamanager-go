import { FundingComparisonChart } from '../funding-comparison-chart';
import { renderWithProviders } from '@/test-utils';
import type { FinancialResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('@nivo/bar', () => ({
  ResponsiveBar: () => <div data-testid="nivo-bar" />,
}));

const emptyData: FinancialResponse = {
  data_points: [],
};

const dataWithActual: FinancialResponse = {
  data_points: [
    {
      date: '2025-01-01',
      funding_income: 500000,
      budget_income: 0,
      gross_salary: 0,
      employer_costs: 0,
      budget_expenses: 0,
      total_income: 500000,
      total_expenses: 0,
      balance: 500000,
      actual_funding: 480000,
      child_count: 20,
      staff_count: 5,
    },
    {
      date: '2025-02-01',
      funding_income: 510000,
      budget_income: 0,
      gross_salary: 0,
      employer_costs: 0,
      budget_expenses: 0,
      total_income: 510000,
      total_expenses: 0,
      balance: 510000,
      child_count: 21,
      staff_count: 5,
    },
  ],
};

const dataWithoutActual: FinancialResponse = {
  data_points: [
    {
      date: '2025-01-01',
      funding_income: 500000,
      budget_income: 0,
      gross_salary: 0,
      employer_costs: 0,
      budget_expenses: 0,
      total_income: 500000,
      total_expenses: 0,
      balance: 500000,
      child_count: 20,
      staff_count: 5,
    },
  ],
};

describe('FundingComparisonChart', () => {
  it('renders without crashing with empty data', () => {
    const { container } = renderWithProviders(<FundingComparisonChart data={emptyData} />);
    expect(container).toBeTruthy();
  });

  it('renders chart when actual funding data exists', () => {
    const { container } = renderWithProviders(<FundingComparisonChart data={dataWithActual} />);
    expect(container).toBeTruthy();
  });

  it('renders chart even when no actual funding exists', () => {
    const { getByTestId } = renderWithProviders(
      <FundingComparisonChart data={dataWithoutActual} />
    );
    expect(getByTestId('nivo-bar')).toBeTruthy();
  });
});
