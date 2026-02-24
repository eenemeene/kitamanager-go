import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ChildrenPrintPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getStaffingHours: jest.fn(),
    getAgeDistribution: jest.fn(),
    getContractPropertiesDistribution: jest.fn(),
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

const mockStaffingData = {
  data_points: [
    {
      date: '2026-01-01',
      required_hours: 312,
      available_hours: 340,
      child_count: 45,
      staff_count: 12,
    },
  ],
};

const mockAgeDistribution = {
  date: '2026-01-01',
  distribution: [{ age: 3, count: 10 }],
  total: 10,
};

const mockContractProperties = {
  date: '2026-01-01',
  properties: [{ key: 'care_type', value: 'ganztag', label: 'Full-Time', count: 30 }],
};

describe('ChildrenPrintPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue(mockStaffingData);
    (apiClient.getAgeDistribution as jest.Mock).mockResolvedValue(mockAgeDistribution);
    (apiClient.getContractPropertiesDistribution as jest.Mock).mockResolvedValue(
      mockContractProperties
    );
    window.print = jest.fn();
  });

  it('renders page title', () => {
    renderWithProviders(<ChildrenPrintPage />);
    expect(screen.getByText('nav.statisticsChildren')).toBeInTheDocument();
  });

  it('renders organization name', () => {
    renderWithProviders(<ChildrenPrintPage />);
    expect(screen.getByText(/Test Kita/)).toBeInTheDocument();
  });

  it('renders print button', () => {
    renderWithProviders(<ChildrenPrintPage />);
    expect(screen.getByText('common.print')).toBeInTheDocument();
  });

  it('calls window.print when print button clicked', async () => {
    renderWithProviders(<ChildrenPrintPage />);
    await userEvent.click(screen.getByText('common.print'));
    expect(window.print).toHaveBeenCalled();
  });

  it('renders children contract count section', async () => {
    renderWithProviders(<ChildrenPrintPage />);
    await waitFor(() => {
      expect(screen.getByText('statistics.childrenContractCount')).toBeInTheDocument();
    });
  });

  it('renders age distribution section', async () => {
    renderWithProviders(<ChildrenPrintPage />);
    await waitFor(() => {
      expect(screen.getByText('statistics.ageDistribution')).toBeInTheDocument();
    });
  });

  it('renders contract properties section', async () => {
    renderWithProviders(<ChildrenPrintPage />);
    await waitFor(() => {
      expect(screen.getByText('statistics.contractProperties')).toBeInTheDocument();
    });
  });

  it('has no sidebar or header elements', () => {
    const { container } = renderWithProviders(<ChildrenPrintPage />);
    expect(container.querySelector('[class*="sidebar"]')).toBeNull();
    expect(container.querySelector('header')).toBeNull();
  });
});
