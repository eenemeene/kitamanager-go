import { FundingComparisonChart } from '../funding-comparison-chart';
import { renderWithProviders } from '@/test-utils';
import type { FinancialResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('@nivo/bar', () => ({
  ResponsiveBar: () => <div data-testid="nivo-bar" />,
}));

function makeDataPoint(
  overrides: Partial<FinancialResponse['data_points'][0]>
): FinancialResponse['data_points'][0] {
  return {
    date: '2025-01-01',
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

const dataWithActual: FinancialResponse = {
  data_points: [
    makeDataPoint({
      date: '2025-01-01',
      funding_income: 500000,
      total_income: 500000,
      balance: 500000,
      actual_funding: 480000,
      actual_funding_regular: 470000,
      actual_funding_correction: 10000,
      child_count: 20,
      staff_count: 5,
    }),
    makeDataPoint({
      date: '2025-02-01',
      funding_income: 510000,
      total_income: 510000,
      balance: 510000,
      child_count: 21,
      staff_count: 5,
    }),
  ],
  warnings: [],
};

const dataWithoutActual: FinancialResponse = {
  data_points: [
    makeDataPoint({
      date: '2025-01-01',
      funding_income: 500000,
      total_income: 500000,
      balance: 500000,
      child_count: 20,
      staff_count: 5,
    }),
  ],
  warnings: [],
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

  it('renders table with Regular, Corrections, and Difference column headers', () => {
    const { container } = renderWithProviders(<FundingComparisonChart data={dataWithActual} />);
    const headerHTML = container.querySelector('thead')?.innerHTML ?? '';
    expect(headerHTML).toContain('fundingActualRegular');
    expect(headerHTML).toContain('fundingActualCorrection');
    expect(headerHTML).toContain('fundingDifference');
  });
});

// B1: the Kita-year row must be reproducible from the cells it shows. The
// Calculated cell therefore carries the bill-months figure the Difference is
// built from, with the full-range total demoted to a sub-line.
describe('FundingComparisonChart Kita year row basis', () => {
  const partiallyBilled: FinancialResponse = {
    data_points: [
      makeDataPoint({
        date: '2025-08-01',
        funding_income: 100000,
        actual_funding: 110000,
        actual_funding_regular: 110000,
        actual_funding_correction: 0,
      }),
      makeDataPoint({
        date: '2025-09-01',
        funding_income: 900000,
        actual_funding: undefined,
        actual_funding_regular: undefined,
        actual_funding_correction: undefined,
      }),
    ],
    warnings: [],
  };

  it('shows the full-range total as a sub-line when the year is only partly billed', () => {
    const { container } = renderWithProviders(<FundingComparisonChart data={partiallyBilled} />);
    expect(container.textContent).toContain('fundingCalculatedFullYear');
  });

  it('omits the sub-line when every month of the year is billed', () => {
    const fullyBilled: FinancialResponse = {
      data_points: [
        makeDataPoint({
          date: '2025-08-01',
          funding_income: 100000,
          actual_funding: 110000,
          actual_funding_regular: 110000,
          actual_funding_correction: 0,
        }),
      ],
      warnings: [],
    };
    const { container } = renderWithProviders(<FundingComparisonChart data={fullyBilled} />);
    expect(container.textContent).not.toContain('fundingCalculatedFullYear');
  });

  it('omits the sub-line when the year has no bills at all', () => {
    const { container } = renderWithProviders(<FundingComparisonChart data={dataWithoutActual} />);
    expect(container.textContent).not.toContain('fundingCalculatedFullYear');
  });
});
