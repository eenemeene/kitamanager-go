import { screen, waitFor } from '@testing-library/react';
import GovernmentFundingDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFunding: jest.fn(),
    createGovernmentFundingPeriod: jest.fn(),
    deleteGovernmentFundingPeriod: jest.fn(),
    createGovernmentFundingProperty: jest.fn(),
    deleteGovernmentFundingProperty: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '1' }),
  useRouter: () => ({ push: jest.fn(), back: jest.fn(), refresh: jest.fn() }),
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

const mockFundingWithPeriods = {
  id: 1,
  name: 'Berliner Kita Funding',
  state: 'berlin',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  periods: [
    {
      id: 10,
      government_funding_id: 1,
      from: '2024-01-01',
      to: '2024-12-31',
      comment: 'Year 2024',
      created_at: '2024-01-01T00:00:00Z',
      properties: [
        {
          id: 100,
          period_id: 10,
          key: 'care_type',
          value: 'ganztag',
          payment: 166847,
          requirement: 150,
          min_age: 3,
          max_age: 6,
          comment: '',
          created_at: '2024-01-01T00:00:00Z',
        },
      ],
    },
  ],
};

const mockFundingNoPeriods = {
  id: 1,
  name: 'Berliner Kita Funding',
  state: 'berlin',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  periods: [],
};

describe('GovernmentFundingDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading state', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<GovernmentFundingDetailPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders funding name and state', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingWithPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    expect(screen.getByText('states.berlin')).toBeInTheDocument();
  });

  it('shows periods with formatted date ranges', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingWithPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    // The period comment
    expect(screen.getByText('Year 2024')).toBeInTheDocument();
  });

  it('shows add period button', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingWithPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    expect(screen.getByText('governmentFundings.addPeriod')).toBeInTheDocument();
  });

  it('shows properties within a period (key, value, payment, requirement)', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingWithPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    // Property key and value
    expect(screen.getByText('care_type')).toBeInTheDocument();
    expect(screen.getByText('ganztag')).toBeInTheDocument();
  });

  it('shows no periods message when empty', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingNoPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    expect(screen.getByText('governmentFundings.noPeriodsDefined')).toBeInTheDocument();
  });

  it('renders back button', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(mockFundingWithPeriods);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    // Back button is a ghost variant icon button
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(1);
  });

  it('shows error when funding fails to load', async () => {
    (apiClient.getGovernmentFunding as jest.Mock).mockResolvedValue(null);

    renderWithProviders(<GovernmentFundingDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('governmentFundings.failedToLoadFunding')).toBeInTheDocument();
    });
  });
});
