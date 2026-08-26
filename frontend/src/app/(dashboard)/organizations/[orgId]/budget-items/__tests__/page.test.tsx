import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import BudgetItemsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
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

  // Budget items feed the expense side of Financials and Forecast. An operator
  // who never creates any gets plausible-looking figures that omit every
  // non-salary cost the Kita has, so the empty list has to explain itself
  // rather than render a bare "no results" row.
  it('shows the onboarding empty state when no budget items exist', async () => {
    renderWithProviders(<BudgetItemsPage />);

    await waitFor(() => {
      expect(screen.getByText('budgetItems.emptyTitle')).toBeInTheDocument();
    });
    expect(screen.getByText('budgetItems.emptyDescription')).toBeInTheDocument();
    expect(screen.queryByText('common.noResults')).not.toBeInTheDocument();
  });

  // With a search active an empty list means "nothing matched", which the user
  // resolves by changing the search — the onboarding panel would be wrong.
  it('falls back to the plain no-results row once a search is active', async () => {
    renderWithProviders(<BudgetItemsPage />);
    await waitFor(() => {
      expect(screen.getByText('budgetItems.emptyTitle')).toBeInTheDocument();
    });

    await userEvent.type(screen.getByLabelText('common.search'), 'nothing matches this');

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });
    expect(screen.queryByText('budgetItems.emptyTitle')).not.toBeInTheDocument();
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
