import { screen, waitFor } from '@testing-library/react';
import { AttendanceSummary } from '../attendance-summary';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChildAttendanceSummary: jest.fn(),
  },
}));

describe('AttendanceSummary', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders loading skeletons', () => {
    (apiClient.getChildAttendanceSummary as jest.Mock).mockImplementation(
      () => new Promise(() => {})
    );
    const { container } = renderWithProviders(<AttendanceSummary orgId={1} date="2024-01-15" />);
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0);
  });

  it('renders summary values', async () => {
    (apiClient.getChildAttendanceSummary as jest.Mock).mockResolvedValue({
      present: 10,
      absent: 2,
      sick: 1,
      vacation: 0,
      total_children: 15,
    });
    renderWithProviders(<AttendanceSummary orgId={1} date="2024-01-15" />);
    await waitFor(() => {
      expect(screen.getByText('10')).toBeInTheDocument();
      expect(screen.getAllByText('2').length).toBeGreaterThanOrEqual(1);
    });
  });
});
