import { screen, waitFor } from '@testing-library/react';
import { StepPromotionsWidget } from '../step-promotions-widget';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getStepPromotions: jest.fn(),
  },
}));

describe('StepPromotionsWidget', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders nothing when no data', async () => {
    (apiClient.getStepPromotions as jest.Mock).mockResolvedValue({
      promotions: [],
      total_monthly_cost_delta: 0,
    });
    const { container } = renderWithProviders(<StepPromotionsWidget orgId={1} />);
    await waitFor(() => {
      expect(apiClient.getStepPromotions).toHaveBeenCalledWith(1);
    });
    // Widget returns null when empty
    expect(container.innerHTML).toBe('');
  });

  it('renders promotions table when data exists', async () => {
    (apiClient.getStepPromotions as jest.Mock).mockResolvedValue({
      total_monthly_cost_delta: 15000,
      promotions: [
        {
          employee_id: 1,
          employee_name: 'Jane Doe',
          grade: 'S8a',
          current_step: 3,
          eligible_step: 4,
          service_start: '2020-03-15T00:00:00Z',
          years_of_service: 6.0,
          current_amount: 300000,
          new_amount: 310000,
          monthly_cost_delta: 15000,
        },
      ],
    });

    renderWithProviders(<StepPromotionsWidget orgId={1} />);

    await waitFor(() => {
      expect(screen.getByText('Jane Doe')).toBeInTheDocument();
    });
    expect(screen.getByText('S8a')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('4')).toBeInTheDocument();
  });
});
