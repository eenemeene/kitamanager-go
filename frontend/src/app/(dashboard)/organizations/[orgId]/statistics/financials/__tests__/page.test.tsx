import { screen } from '@testing-library/react';
import FinancialsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('next/dynamic', () => () => {
  const DynamicComponent = () => <div data-testid="dynamic-chart" />;
  DynamicComponent.displayName = 'DynamicComponent';
  return DynamicComponent;
});

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getFinancials: jest.fn(),
    getBudgetItems: jest.fn(),
  },
}));

describe('FinancialsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getFinancials as jest.Mock).mockResolvedValue({ data_points: [] });
    (apiClient.getBudgetItems as jest.Mock).mockResolvedValue({ data: [] });
  });

  it('renders the page', () => {
    renderWithProviders(<FinancialsPage />);
    expect(screen.getByText('nav.statisticsFinancials')).toBeInTheDocument();
  });
});
