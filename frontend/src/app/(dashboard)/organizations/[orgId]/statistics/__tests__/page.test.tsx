import { screen } from '@testing-library/react';
import StatisticsPage from '../page';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

describe('StatisticsPage (Overview)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title', () => {
    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('statistics.title')).toBeInTheDocument();
  });

  it('renders sub-page link cards', () => {
    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('nav.statisticsFinancials')).toBeInTheDocument();
    expect(screen.getByText('nav.statisticsStaffing')).toBeInTheDocument();
    expect(screen.getByText('nav.statisticsChildren')).toBeInTheDocument();
  });

  it('renders links to sub-pages with correct hrefs', () => {
    renderWithProviders(<StatisticsPage />);

    const links = screen.getAllByRole('link');
    const hrefs = links.map((link: HTMLElement) => link.getAttribute('href'));

    expect(hrefs).toContain('/organizations/1/statistics/financials');
    expect(hrefs).toContain('/organizations/1/statistics/staffing');
    expect(hrefs).toContain('/organizations/1/statistics/children');
    expect(hrefs).toContain('/organizations/1/statistics/budget');
  });

  it('renders budget link card', () => {
    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('nav.statisticsBudget')).toBeInTheDocument();
  });

  it('renders forecast link card', () => {
    renderWithProviders(<StatisticsPage />);

    expect(screen.getByText('nav.statisticsForecast')).toBeInTheDocument();
  });
});
