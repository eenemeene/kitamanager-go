import { screen } from '@testing-library/react';
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
    (apiClient.getChildAttendanceSummary as jest.Mock).mockResolvedValue({ summary: [] });
  });

  it('renders the attendance page', () => {
    renderWithProviders(<AttendancePage />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('title');
  });
});
