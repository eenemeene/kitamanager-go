import { screen } from '@testing-library/react';
import GovernmentFundingBillDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', id: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFundingBillPeriod: jest.fn(),
    compareGovernmentFundingBill: jest.fn(),
  },
}));

describe('GovernmentFundingBillDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getGovernmentFundingBillPeriod as jest.Mock).mockImplementation(
      () => new Promise(() => {})
    );
    (apiClient.compareGovernmentFundingBill as jest.Mock).mockImplementation(
      () => new Promise(() => {})
    );
  });

  it('renders loading state', () => {
    renderWithProviders(<GovernmentFundingBillDetailPage />);
    expect(screen.getByText('loading')).toBeInTheDocument();
  });
});
