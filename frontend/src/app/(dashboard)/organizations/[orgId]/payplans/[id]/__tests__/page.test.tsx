import PayplanDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', id: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/components/charts/payplan-salary-chart', () => ({
  PayPlanSalaryChart: () => <div data-testid="salary-chart" />,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getPayPlan: jest.fn(),
    getPayPlanExportUrl: jest.fn().mockReturnValue('/export'),
    createPayPlanPeriod: jest.fn(),
    updatePayPlanPeriod: jest.fn(),
    deletePayPlanPeriod: jest.fn(),
    createPayPlanEntry: jest.fn(),
    updatePayPlanEntry: jest.fn(),
    deletePayPlanEntry: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('PayplanDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getPayPlan as jest.Mock).mockImplementation(() => new Promise(() => {}));
  });

  it('renders loading state', () => {
    const { container } = renderWithProviders(<PayplanDetailPage />);
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });
});
