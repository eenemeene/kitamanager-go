import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import OccupancyPrintPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getOccupancy: jest.fn(),
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

const mockOccupancyData = {
  age_groups: [{ label: '0-3', min_age: 0, max_age: 3 }],
  care_types: [{ value: 'ganztag', label: 'Full-Time' }],
  supplement_types: [],
  data_points: [
    {
      date: '2026-01-01',
      total: 45,
      by_age_and_care_type: { '0-3': { ganztag: 10 } },
      by_supplement: {},
    },
  ],
};

describe('OccupancyPrintPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getOccupancy as jest.Mock).mockResolvedValue(mockOccupancyData);
    window.print = jest.fn();
  });

  it('renders page title', () => {
    renderWithProviders(<OccupancyPrintPage />);
    expect(screen.getByText('nav.statisticsOccupancy')).toBeInTheDocument();
  });

  it('renders organization name', () => {
    renderWithProviders(<OccupancyPrintPage />);
    expect(screen.getByText(/Test Kita/)).toBeInTheDocument();
  });

  it('renders print button', () => {
    renderWithProviders(<OccupancyPrintPage />);
    expect(screen.getByText('common.print')).toBeInTheDocument();
  });

  it('calls window.print when print button clicked', async () => {
    renderWithProviders(<OccupancyPrintPage />);
    await userEvent.click(screen.getByText('common.print'));
    expect(window.print).toHaveBeenCalled();
  });

  it('renders occupancy matrix section', async () => {
    renderWithProviders(<OccupancyPrintPage />);
    await waitFor(() => {
      expect(screen.getByText('statistics.occupancyMatrix')).toBeInTheDocument();
    });
  });

  it('has no sidebar or header elements', () => {
    const { container } = renderWithProviders(<OccupancyPrintPage />);
    expect(container.querySelector('[class*="sidebar"]')).toBeNull();
    expect(container.querySelector('header')).toBeNull();
  });
});
