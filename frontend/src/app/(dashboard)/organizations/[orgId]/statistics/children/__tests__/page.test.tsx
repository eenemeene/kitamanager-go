import { screen } from '@testing-library/react';
import ChildrenStatisticsPage from '../page';
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
    getAgeDistribution: jest.fn(),
    getContractPropertiesDistribution: jest.fn(),
    getStaffingHours: jest.fn(),
    getSections: jest.fn(),
  },
}));

describe('ChildrenStatisticsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getAgeDistribution as jest.Mock).mockResolvedValue([]);
    (apiClient.getContractPropertiesDistribution as jest.Mock).mockResolvedValue([]);
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue({ data_points: [] });
    (apiClient.getSections as jest.Mock).mockResolvedValue({ data: [] });
  });

  it('renders the page title', () => {
    renderWithProviders(<ChildrenStatisticsPage />);
    expect(screen.getByText('nav.statisticsChildren')).toBeInTheDocument();
  });
});
