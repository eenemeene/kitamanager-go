import { screen, waitFor } from '@testing-library/react';
import StatisticsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getFinancials: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

const mockFinancials = {
  data_points: [
    {
      date: `${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, '0')}-01`,
      total_income: 500000,
      total_expenses: 300000,
      balance: 200000,
      funding_income: 500000,
      gross_salary: 200000,
      employer_costs: 50000,
      budget_income: 0,
      budget_expenses: 0,
    },
  ],
};

describe('StatisticsPage (Overview)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title', async () => {
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancials);

    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('statistics.title')).toBeInTheDocument();
  });

  it('renders sub-page link cards', async () => {
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancials);

    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('nav.statisticsFinancials')).toBeInTheDocument();
    expect(screen.getByText('nav.statisticsStaffing')).toBeInTheDocument();
    expect(screen.getByText('nav.statisticsChildren')).toBeInTheDocument();
  });

  it('renders financial summary cards when data is loaded', async () => {
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancials);

    renderWithProviders(<StatisticsPage />);

    await waitFor(() => {
      expect(screen.getByText('statistics.totalIncome')).toBeInTheDocument();
    });

    expect(screen.getByText('statistics.totalExpenses')).toBeInTheDocument();
    expect(screen.getByText('statistics.balance')).toBeInTheDocument();
  });

  it('shows loading skeletons while fetching', async () => {
    (apiClient.getFinancials as jest.Mock).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<StatisticsPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders links to sub-pages with correct hrefs', async () => {
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancials);

    renderWithProviders(<StatisticsPage />);

    const links = screen.getAllByRole('link');
    const hrefs = links.map((link) => link.getAttribute('href'));

    expect(hrefs).toContain('/organizations/1/statistics/financials');
    expect(hrefs).toContain('/organizations/1/statistics/staffing');
    expect(hrefs).toContain('/organizations/1/statistics/children');
  });
});
