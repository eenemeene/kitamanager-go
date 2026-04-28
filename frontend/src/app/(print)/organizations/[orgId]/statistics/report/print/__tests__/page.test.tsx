import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CombinedReportPrintPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    // Cover-level
    getChildren: jest.fn(),
    getEmployees: jest.fn(),
    // Section-level (each section's queries):
    getStaffingHours: jest.fn(),
    getEmployeeStaffingHours: jest.fn(),
    getOccupancy: jest.fn(),
    getAgeDistribution: jest.fn(),
    getContractPropertiesDistribution: jest.fn(),
    getSections: jest.fn(),
    getFinancials: jest.fn(),
    compareBills: jest.fn(),
  },
  getErrorMessage: jest.fn((_e, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
  useSearchParams: () => new URLSearchParams('month=2026-04'),
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

const mockStaffingData = {
  data_points: [
    {
      date: '2026-04-01',
      required_hours: 312,
      available_hours: 340,
      child_count: 45,
      staff_count: 12,
    },
  ],
};

const mockFinancialData = {
  data_points: [
    {
      date: '2026-04-01',
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

describe('CombinedReportPrintPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getChildren as jest.Mock).mockResolvedValue({ data: [], total: 45 });
    (apiClient.getEmployees as jest.Mock).mockResolvedValue({ data: [], total: 12 });
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue(mockStaffingData);
    (apiClient.getEmployeeStaffingHours as jest.Mock).mockResolvedValue({
      employees: [],
      months: [],
    });
    (apiClient.getOccupancy as jest.Mock).mockResolvedValue({
      age_groups: [],
      care_types: [],
      supplement_types: [],
      data_points: [],
    });
    (apiClient.getAgeDistribution as jest.Mock).mockResolvedValue({
      date: '2026-04-01',
      distribution: [],
      total: 0,
    });
    (apiClient.getContractPropertiesDistribution as jest.Mock).mockResolvedValue({
      date: '2026-04-01',
      properties: [],
    });
    (apiClient.getSections as jest.Mock).mockResolvedValue({ data: [], total: 0 });
    (apiClient.getFinancials as jest.Mock).mockResolvedValue(mockFinancialData);
    (apiClient.compareBills as jest.Mock).mockResolvedValue({
      comparisons: [],
      summary: undefined,
    });
    window.print = jest.fn();
  });

  it('renders the cover with the organization name', () => {
    renderWithProviders(<CombinedReportPrintPage />);
    // Org name appears twice: once in the no-print header, once in the cover.
    expect(screen.getAllByText(/Test Kita/).length).toBeGreaterThan(0);
  });

  it('renders the print action button (no-print but in DOM)', () => {
    renderWithProviders(<CombinedReportPrintPage />);
    expect(screen.getByText('report.printAction')).toBeInTheDocument();
  });

  it('triggers window.print when the action is clicked', async () => {
    renderWithProviders(<CombinedReportPrintPage />);
    await userEvent.click(screen.getByText('report.printAction'));
    expect(window.print).toHaveBeenCalled();
  });

  it('renders all four report sections (children/occupancy/staffing/financials chart titles appear)', async () => {
    renderWithProviders(<CombinedReportPrintPage />);
    await waitFor(() => {
      // each section emits at least one section-specific i18n key
      expect(screen.getByText('statistics.childrenContractCount')).toBeInTheDocument();
      expect(screen.getByText('statistics.occupancyMatrix')).toBeInTheDocument();
      expect(screen.getByText('statistics.staffingHours')).toBeInTheDocument();
      expect(screen.getByText('statistics.financialOverview')).toBeInTheDocument();
    });
  });

  it('fires the same query keys as the standalone sections (TanStack dedup)', async () => {
    renderWithProviders(<CombinedReportPrintPage />);
    await waitFor(() => {
      // Cover + staffing section both want the kita-year staffing-hours
      // window — only one network call should result.
      const kitaYearCalls = (apiClient.getStaffingHours as jest.Mock).mock.calls.filter(
        ([, opts]) => opts?.from === '2025-08-01' && opts?.to === '2026-07-01'
      );
      expect(kitaYearCalls.length).toBe(1);
    });
  });
});
