import { screen, waitFor } from '@testing-library/react';
import { SectionAgeAlertsWidget } from '../section-age-alerts-widget';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChildrenAll: jest.fn(),
    getSections: jest.fn(),
    updateChildContract: jest.fn(),
  },
}));

describe('SectionAgeAlertsWidget', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders nothing when no children', async () => {
    (apiClient.getChildrenAll as jest.Mock).mockResolvedValue([]);
    (apiClient.getSections as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
    const { container } = renderWithProviders(<SectionAgeAlertsWidget orgId={1} />);
    await waitFor(() => {
      expect(apiClient.getChildrenAll).toHaveBeenCalled();
    });
    expect(container.innerHTML).toBe('');
  });

  it('shows alerts for children exceeding section age', async () => {
    // Child born 4 years ago, section max is 36 months
    const fourYearsAgo = new Date(Date.now() - 4 * 365.25 * 86400000).toISOString();
    (apiClient.getChildrenAll as jest.Mock).mockResolvedValue([
      {
        id: 1,
        first_name: 'Lena',
        last_name: 'Test',
        birthdate: fourYearsAgo,
        contracts: [{ id: 10, from: '2022-01-01', section_id: 100 }],
      },
    ]);
    (apiClient.getSections as jest.Mock).mockResolvedValue(
      createMockPaginatedResponse([
        { id: 100, name: 'Krippe', min_age_months: 0, max_age_months: 36 },
        { id: 200, name: 'Kita', min_age_months: 36, max_age_months: 72 },
      ])
    );

    renderWithProviders(<SectionAgeAlertsWidget orgId={1} />);

    await waitFor(() => {
      expect(screen.getByText('Lena Test')).toBeInTheDocument();
    });
    expect(screen.getByText('Krippe')).toBeInTheDocument();
  });
});
