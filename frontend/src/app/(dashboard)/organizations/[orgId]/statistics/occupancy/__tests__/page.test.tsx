import { screen } from '@testing-library/react';
import OccupancyPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('next-intl', () => ({
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
