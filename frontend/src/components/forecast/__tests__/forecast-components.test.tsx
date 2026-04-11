import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test-utils';
import { ForecastResults } from '../forecast-results';
import { ForecastModificationSummary } from '../forecast-modification-summary';
import type { ForecastResponse } from '@/lib/api/types';

// Mock next-intl
jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
  usePathname: () => '/organizations/1/statistics/forecast',
}));

// Mock next/dynamic to just render children
jest.mock('next/dynamic', () => () => {
  function DynamicComponent() {
    return <div data-testid="dynamic-chart">Chart</div>;
  }
  return DynamicComponent;
});

// Mock API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn().mockResolvedValue({ data: [], total: 0 }),
    getChildrenAll: jest.fn().mockResolvedValue([]),
    getEmployeesAll: jest.fn().mockResolvedValue([]),
    getPayPlans: jest.fn().mockResolvedValue({ data: [], total: 0 }),
    getBudgetItems: jest.fn().mockResolvedValue({ data: [], total: 0 }),
    getFinancials: jest.fn().mockResolvedValue({ data_points: [] }),
    getStaffingHours: jest.fn().mockResolvedValue({ data_points: [] }),
    getOccupancy: jest.fn().mockResolvedValue({ data: [] }),
    getEmployeeStaffingHours: jest.fn().mockResolvedValue({ dates: [], employees: [] }),
    postForecast: jest.fn().mockResolvedValue({}),
  },
}));

// Mock the funding attributes hook
jest.mock('@/lib/hooks/use-funding-attributes', () => ({
  useFundingAttributes: () => ({
    fundingAttributes: [],
    attributesByKey: {},
    defaultProperties: undefined,
  }),
}));

describe('ForecastResults', () => {
  const mockData: ForecastResponse = {
    financials: {
      data_points: [
        {
          date: '2026-01-01',
          total_income: 100000,
          total_expenses: 80000,
          balance: 20000,
          funding_income: 90000,
          gross_salary: 60000,
          employer_costs: 15000,
          budget_income: 10000,
          budget_expenses: 5000,
          child_count: 20,
          staff_count: 5,
        },
      ],
    },
    staffing_hours: {
      data_points: [
        {
          date: '2026-01-01',
          required_hours: 200,
          available_hours: 180,
          child_count: 20,
          staff_count: 5,
        },
      ],
    },
    occupancy: {
      age_groups: [],
      care_types: [],
      supplement_types: [],
      data_points: [],
    },
    employee_staffing_hours: {
      dates: ['2026-01-01'],
      employees: [],
    },
  };

  it('renders results card with tabs', () => {
    renderWithProviders(<ForecastResults data={mockData} />);
    expect(screen.getByText('forecastResults')).toBeInTheDocument();
    expect(screen.getByText('forecastTabFinancials')).toBeInTheDocument();
    expect(screen.getByText('forecastTabStaffing')).toBeInTheDocument();
    expect(screen.getByText('forecastTabOccupancy')).toBeInTheDocument();
    expect(screen.getByText('forecastTabEmployeeHours')).toBeInTheDocument();
  });

  it('shows baseline toggle when baseline is provided', () => {
    renderWithProviders(
      <ForecastResults
        data={mockData}
        baseline={{
          financials: mockData.financials,
          staffingHours: mockData.staffing_hours,
          occupancy: mockData.occupancy,
          employeeStaffingHours: mockData.employee_staffing_hours,
          isLoading: false,
        }}
      />
    );
    expect(screen.getByText('forecastShowBaseline')).toBeInTheDocument();
  });

  it('does not show baseline toggle when no baseline', () => {
    renderWithProviders(<ForecastResults data={mockData} />);
    expect(screen.queryByText('forecastShowBaseline')).not.toBeInTheDocument();
  });
});

describe('ForecastModificationSummary', () => {
  beforeEach(() => {
    // Reset the store between tests
    const { useForecastStore } = require('@/stores/forecast-store');
    useForecastStore.getState().reset();
  });

  it('renders nothing when no modifications', () => {
    const { container } = renderWithProviders(<ForecastModificationSummary />);
    expect(container.innerHTML).toBe('');
  });

  it('renders badges when modifications exist', () => {
    const { useForecastStore } = require('@/stores/forecast-store');
    useForecastStore.getState().addChild({
      first_name: 'Child',
      last_name: '#1',
      gender: 'diverse',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    renderWithProviders(<ForecastModificationSummary />);
    expect(screen.getByText(/forecastAddChild: 1/)).toBeInTheDocument();
  });
});
