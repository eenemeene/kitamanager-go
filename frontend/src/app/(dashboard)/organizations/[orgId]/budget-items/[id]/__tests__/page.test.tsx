import { screen } from '@testing-library/react';
import BudgetItemDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', id: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getBudgetItem: jest.fn(),
    createBudgetItemEntry: jest.fn(),
    updateBudgetItemEntry: jest.fn(),
    deleteBudgetItemEntry: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('BudgetItemDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getBudgetItem as jest.Mock).mockResolvedValue({
      id: 1,
      name: 'Rent',
      category: 'expense',
      per_child: false,
      entries: [],
    });
  });

  it('renders loading state initially', () => {
    (apiClient.getBudgetItem as jest.Mock).mockImplementation(() => new Promise(() => {}));
    const { container } = renderWithProviders(<BudgetItemDetailPage />);
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });
});
