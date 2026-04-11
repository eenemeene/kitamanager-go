import { renderWithProviders } from '@/test-utils';
import { ForecastChildrenTab } from '../forecast-children-tab';
import { ForecastEmployeesTab } from '../forecast-employees-tab';
import { ForecastOptimizeTab } from '../forecast-optimize-tab';

// Mock next-intl
jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
  usePathname: () => '/organizations/1/statistics/forecast',
}));

// Mock API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn().mockResolvedValue({ data: [], total: 0 }),
    getChildrenAll: jest.fn().mockResolvedValue([]),
    getEmployeesAll: jest.fn().mockResolvedValue([]),
    getPayPlans: jest.fn().mockResolvedValue({ data: [], total: 0 }),
  },
}));

// Mock the funding attributes hook
jest.mock('@/lib/hooks/use-funding-attributes', () => ({
  useFundingAttributes: () => ({
    fundingAttributes: [],
    attributesByKey: {},
    defaultProperties: undefined,
  }),
}));

describe('ForecastChildrenTab', () => {
  it('renders without crashing', () => {
    const { container } = renderWithProviders(<ForecastChildrenTab />);
    expect(container.innerHTML).not.toBe('');
  });
});

describe('ForecastEmployeesTab', () => {
  it('renders without crashing', () => {
    const { container } = renderWithProviders(<ForecastEmployeesTab />);
    expect(container.innerHTML).not.toBe('');
  });
});

describe('ForecastOptimizeTab', () => {
  it('renders without crashing', () => {
    const { container } = renderWithProviders(<ForecastOptimizeTab />);
    expect(container.innerHTML).not.toBe('');
  });
});
