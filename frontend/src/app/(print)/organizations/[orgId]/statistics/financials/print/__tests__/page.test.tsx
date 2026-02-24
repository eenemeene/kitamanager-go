import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import FinancialsPrintPage from '../page';
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
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/stores/ui-store', () => ({
  useUiStore: () => ({
    organizations: [{ id: 1, name: 'Test Kita' }],
    fetchOrganizations: jest.fn(),
  }),
}));

const mockFinancialData = {
  data_points: [
    {
      date: '2026-01-01',
      funding_income: 500000,
      gross_salary: 300000,
      employer_costs: 60000,
      budget_income: 20000,
      budget_expenses: 10000,
      total_income: 520000,
      total_expenses: 370000,
      balance: 150000,
      child_count: 45,
      staff_count: 12,
    },
  ],
};

describe('FinancialsPrintPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancialData);
    window.print = jest.fn();
  });

  it('renders page title', () => {
    renderWithProviders(<FinancialsPrintPage />);
    expect(screen.getByText('nav.statisticsFinancials')).toBeInTheDocument();
  });

  it('renders organization name', () => {
    renderWithProviders(<FinancialsPrintPage />);
    expect(screen.getByText(/Test Kita/)).toBeInTheDocument();
  });

  it('renders print button', () => {
    renderWithProviders(<FinancialsPrintPage />);
    expect(screen.getByText('common.print')).toBeInTheDocument();
  });

  it('calls window.print when print button clicked', async () => {
    renderWithProviders(<FinancialsPrintPage />);
    await userEvent.click(screen.getByText('common.print'));
    expect(window.print).toHaveBeenCalled();
  });

  it('renders financial summary cards when data loads', async () => {
    renderWithProviders(<FinancialsPrintPage />);
    await waitFor(() => {
      expect(screen.getByText('statistics.totalIncome')).toBeInTheDocument();
      expect(screen.getByText('statistics.totalExpenses')).toBeInTheDocument();
      expect(screen.getByText('statistics.balance')).toBeInTheDocument();
    });
  });

  it('renders budget overview section', async () => {
    renderWithProviders(<FinancialsPrintPage />);
    await waitFor(() => {
      expect(screen.getByText(/statistics.budgetOverview/)).toBeInTheDocument();
    });
  });

  it('has no sidebar or header elements', () => {
    const { container } = renderWithProviders(<FinancialsPrintPage />);
    expect(container.querySelector('[class*="sidebar"]')).toBeNull();
    expect(container.querySelector('header')).toBeNull();
  });
});
