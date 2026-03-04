import { screen } from '@testing-library/react';
import GovernmentFundingBillsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFundingBillPeriods: jest.fn(),
    uploadGovernmentFundingBill: jest.fn(),
    deleteGovernmentFundingBillPeriod: jest.fn(),
    compareGovernmentFundingBill: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('GovernmentFundingBillsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getGovernmentFundingBillPeriods as jest.Mock).mockResolvedValue(
      createMockPaginatedResponse([])
    );
  });

  it('renders the page title', () => {
    renderWithProviders(<GovernmentFundingBillsPage />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('title');
  });
});
