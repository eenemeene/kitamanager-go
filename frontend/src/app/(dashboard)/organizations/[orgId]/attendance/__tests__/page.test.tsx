import { screen, waitFor } from '@testing-library/react';
import AttendancePage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn(),
    getChildren: jest.fn(),
    getChildrenAllForDate: jest.fn(),
    getChildAttendanceByDateAll: jest.fn(),
    getChildAttendanceSummary: jest.fn(),
    createChildAttendance: jest.fn(),
    updateChildAttendance: jest.fn(),
    deleteChildAttendance: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('AttendancePage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getSections as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
    (apiClient.getChildren as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
    (apiClient.getChildrenAllForDate as jest.Mock).mockResolvedValue([]);
    (apiClient.getChildAttendanceByDateAll as jest.Mock).mockResolvedValue([]);
    (apiClient.getChildAttendanceSummary as jest.Mock).mockResolvedValue({ summary: [] });
  });

  it('renders the attendance page', () => {
    renderWithProviders(<AttendancePage />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('title');
  });

  /**
   * The roster is per day, not per week.
   *
   * It used to be fetched once, for the Monday, and reused for all five
   * columns. A child whose contract started on the Tuesday had no row at all --
   * no way to record them -- and one whose contract ended on the Wednesday kept
   * an inviting check-in button for two days they were no longer enrolled for.
   */
  describe('week roster', () => {
    /** The dates the page asked the children endpoint about, in order. */
    async function requestedRosterDates(): Promise<string[]> {
      await waitFor(() =>
        expect((apiClient.getChildrenAllForDate as jest.Mock).mock.calls.length).toBeGreaterThan(0)
      );
      return (apiClient.getChildrenAllForDate as jest.Mock).mock.calls.map((c) => c[1] as string);
    }

    it('asks for every weekday of the displayed week', async () => {
      renderWithProviders(<AttendancePage />);

      const dates = await waitFor(async () => {
        const d = await requestedRosterDates();
        expect(new Set(d).size).toBe(5);
        return d;
      });

      // Monday through Friday, consecutive.
      const sorted = [...new Set(dates)].sort();
      for (let i = 1; i < sorted.length; i++) {
        const previous = new Date(`${sorted[i - 1]}T00:00:00Z`);
        const current = new Date(`${sorted[i]}T00:00:00Z`);
        expect(current.getTime() - previous.getTime()).toBe(24 * 60 * 60 * 1000);
      }
      expect(new Date(`${sorted[0]}T00:00:00Z`).getUTCDay()).toBe(1); // Monday
    });

    it('unions the five days into one row set', async () => {
      // Alice is enrolled all week, Bob only from the Tuesday. Both must have a
      // row; before this, Bob had none.
      const child = (id: number, first: string) => ({
        id,
        organization_id: 1,
        first_name: first,
        last_name: 'Test',
        birthdate: '2021-01-01',
        gender: 'female' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      });
      let call = 0;
      (apiClient.getChildrenAllForDate as jest.Mock).mockImplementation(() => {
        call += 1;
        return Promise.resolve(
          call === 1 ? [child(1, 'Alice')] : [child(1, 'Alice'), child(2, 'Bob')]
        );
      });

      renderWithProviders(<AttendancePage />);

      expect(await screen.findByText('Bob Test')).toBeInTheDocument();
      expect(screen.getByText('Alice Test')).toBeInTheDocument();
    });

    it('marks the days a child is not enrolled for', async () => {
      // The other half of the per-day roster: the union gives Bob a row, and
      // the per-day sets are what stop that row offering a check-in for the
      // Monday he was not enrolled for.
      //
      // Both used to be useMemos whose dependency array held
      // `weekChildrenQueries.map((q) => q.data)` -- a fresh array on every
      // render, so neither memo ever hit and both recomputed constantly behind
      // an exhaustive-deps suppression. They are now derived in `combine`,
      // which react-query memoizes against the query results. This asserts the
      // output survived the move; the recomputation was invisible from outside,
      // which is why it went unnoticed for so long.
      const child = (id: number, first: string) => ({
        id,
        organization_id: 1,
        first_name: first,
        last_name: 'Test',
        birthdate: '2021-01-01',
        gender: 'female' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      });
      // Monday: Alice only. Tuesday onwards: Alice and Bob.
      let call = 0;
      (apiClient.getChildrenAllForDate as jest.Mock).mockImplementation(() => {
        call += 1;
        return Promise.resolve(
          call === 1 ? [child(1, 'Alice')] : [child(1, 'Alice'), child(2, 'Bob')]
        );
      });

      renderWithProviders(<AttendancePage />);

      expect(await screen.findByText('Bob Test')).toBeInTheDocument();
      // Exactly one cell says "not enrolled": Bob's Monday. Alice is enrolled
      // every day, and Bob is enrolled Tuesday to Friday.
      await waitFor(() => expect(screen.getAllByText('notEnrolled')).toHaveLength(1));
    });
  });
});
