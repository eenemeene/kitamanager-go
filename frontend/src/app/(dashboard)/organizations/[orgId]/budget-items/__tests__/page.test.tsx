import { screen } from '@testing-library/react';
import BudgetItemsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
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
    getBudgetItems: jest.fn(),
    createBudgetItem: jest.fn(),
    updateBudgetItem: jest.fn(),
    deleteBudgetItem: jest.fn(),
    createBudgetItemWithEntry: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('BudgetItemsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getBudgetItems as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
  });

  it('renders the page title', () => {
    renderWithProviders(<BudgetItemsPage />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('budgetItems.title');
  });

  it('renders new budget item button', () => {
    renderWithProviders(<BudgetItemsPage />);
    expect(screen.getByText('budgetItems.newBudgetItem')).toBeInTheDocument();
  });
});
