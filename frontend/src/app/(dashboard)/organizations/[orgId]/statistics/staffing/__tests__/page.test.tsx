import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import StaffingPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getStaffingHours: jest.fn(),
    getSections: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
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
    {
      date: '2026-02-01',
      required_hours: 315,
      available_hours: 338,
      child_count: 46,
      staff_count: 12,
    },
  ],
};

const mockSections = {
  data: [
    { id: 1, name: 'Section A', organization_id: 1 },
    { id: 2, name: 'Section B', organization_id: 1 },
  ],
  total: 2,
};

describe('StaffingPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue(mockStaffingData);
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockSections);
  });

  it('renders page title', async () => {
    renderWithProviders(<StaffingPage />);

    expect(screen.getByText('nav.statisticsStaffing')).toBeInTheDocument();
  });

  it('shows loading skeleton while fetching', () => {
    (apiClient.getStaffingHours as jest.Mock).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<StaffingPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders grid table when data loads', async () => {
    renderWithProviders(<StaffingPage />);

    await waitFor(() => {
      expect(screen.getByText('statistics.staffingHoursGrid')).toBeInTheDocument();
    });

    // Check that grid table metric rows are rendered
    await waitFor(() => {
      expect(screen.getByText('staffingRequired')).toBeInTheDocument();
    });
  });

  it('renders year stepper with navigation buttons', () => {
    renderWithProviders(<StaffingPage />);

    expect(screen.getByLabelText('previousYear')).toBeInTheDocument();
    expect(screen.getByLabelText('nextYear')).toBeInTheDocument();
  });

  it('changes query when year stepper is used', async () => {
    renderWithProviders(<StaffingPage />);

    await userEvent.click(screen.getByLabelText('nextYear'));

    await waitFor(() => {
      const calls = (apiClient.getStaffingHours as jest.Mock).mock.calls;
      const gridCalls = calls.filter((c: Array<Record<string, unknown>>) => c[1]?.from && c[1]?.to);
      const lastGridCall = gridCalls[gridCalls.length - 1];
      expect(lastGridCall[1]).toMatchObject({
        from: `${new Date().getFullYear() + 1}-01-01`,
        to: `${new Date().getFullYear() + 1}-12-01`,
      });
    });
  });
});
