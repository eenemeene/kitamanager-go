import { screen, waitFor } from '@testing-library/react';
import { UpcomingChildrenWidget } from '../upcoming-children-widget';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getUpcomingChildren: jest.fn(),
  },
}));

describe('UpcomingChildrenWidget', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders nothing when no upcoming children', async () => {
    (apiClient.getUpcomingChildren as jest.Mock).mockResolvedValue([]);
    const { container } = renderWithProviders(<UpcomingChildrenWidget orgId={1} />);
    await waitFor(() => {
      expect(apiClient.getUpcomingChildren).toHaveBeenCalledWith(1);
    });
    expect(container.innerHTML).toBe('');
  });

  it('renders children table when data exists', async () => {
    const futureDate = new Date(Date.now() + 86400000 * 30).toISOString();
    (apiClient.getUpcomingChildren as jest.Mock).mockResolvedValue([
      {
        id: 1,
        first_name: 'Max',
        last_name: 'Mustermann',
        gender: 'male',
        birthdate: '2022-06-01T00:00:00Z',
        contracts: [{ id: 10, from: futureDate, section_name: 'Krippe', properties: {} }],
      },
    ]);

    renderWithProviders(<UpcomingChildrenWidget orgId={1} />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });
    expect(screen.getByText('Krippe')).toBeInTheDocument();
  });
});
