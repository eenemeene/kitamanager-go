import { screen } from '@testing-library/react';
import OccupancyPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

// The page reads its window from the query string, so useSearchParams has to
// be here too — a file-level mock replaces the one in jest.setup.js wholesale
// rather than extending it.
jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getOccupancy: jest.fn(),
    getSections: jest.fn(),
  },
}));

describe('OccupancyPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getOccupancy as jest.Mock).mockResolvedValue({ sections: [] });
    (apiClient.getSections as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
  });

  it('renders the page title', () => {
    renderWithProviders(<OccupancyPage />);
    expect(screen.getByText('nav.statisticsOccupancy')).toBeInTheDocument();
  });
});
