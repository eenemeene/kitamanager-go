import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test-utils';
import ForecastPage from '../page';

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

describe('ForecastPage', () => {
  it('renders the page with header and config tabs', () => {
    renderWithProviders(<ForecastPage />);
    expect(screen.getByText('statistics.forecastTitle')).toBeInTheDocument();
    expect(screen.getByText('statistics.forecastConfigTitle')).toBeInTheDocument();
    expect(screen.getByText('statistics.forecastTabChildren')).toBeInTheDocument();
    expect(screen.getByText('statistics.forecastCalculate')).toBeInTheDocument();
    expect(screen.getByText('statistics.forecastReset')).toBeInTheDocument();
  });

  it('shows no-results placeholder initially', () => {
    renderWithProviders(<ForecastPage />);
    expect(screen.getByText('statistics.forecastNoResults')).toBeInTheDocument();
  });
});
