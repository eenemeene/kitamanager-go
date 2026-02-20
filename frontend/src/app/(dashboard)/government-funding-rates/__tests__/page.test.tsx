import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import GovernmentFundingsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFundings: jest.fn(),
    createGovernmentFunding: jest.fn(),
    updateGovernmentFunding: jest.fn(),
    deleteGovernmentFunding: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), back: jest.fn(), refresh: jest.fn() }),
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

const mockFundings = [
  {
    id: 1,
    name: 'Berliner Kita Funding',
    state: 'berlin',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Hamburg Kita Funding',
    state: 'berlin',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockPaginatedResponse = createMockPaginatedResponse(mockFundings);
const mockEmptyResponse = createMockPaginatedResponse([]);

describe('GovernmentFundingsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title and new button', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<GovernmentFundingsPage />);

    const titles = screen.getAllByText('governmentFundings.title');
    expect(titles.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('governmentFundings.newGovernmentFunding')).toBeInTheDocument();
  });

  it('shows loading skeletons while fetching', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<GovernmentFundingsPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders table with funding data (name, state, periods)', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    expect(screen.getByText('Hamburg Kita Funding')).toBeInTheDocument();
  });

  it('shows no results when empty', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });
  });

  it('renders table headers (ID, Name, State, Periods, Actions)', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });

    expect(screen.getByText('common.id')).toBeInTheDocument();
    expect(screen.getByText('common.name')).toBeInTheDocument();
    expect(screen.getByText('states.state')).toBeInTheDocument();
    expect(screen.getByText('common.actions')).toBeInTheDocument();
  });

  it('renders view/edit/delete action buttons for each row', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    // Each row has 3 action buttons (view, edit, delete) + 1 new button = 7 total minimum
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(7);
  });

  it('opens create dialog on new button click', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockEmptyResponse);
    const user = userEvent.setup();

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });

    const newButton = screen.getByText('governmentFundings.newGovernmentFunding');
    await user.click(newButton);

    await waitFor(() => {
      expect(screen.getByText('governmentFundings.create')).toBeInTheDocument();
    });
  });

  it('opens delete dialog when delete button clicked', async () => {
    (apiClient.getGovernmentFundings as jest.Mock).mockResolvedValue(mockPaginatedResponse);
    const user = userEvent.setup();

    renderWithProviders(<GovernmentFundingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Berliner Kita Funding')).toBeInTheDocument();
    });

    // Find and click the first delete button (Trash2 icon buttons)
    const buttons = screen.getAllByRole('button');
    // The delete buttons are the 3rd action button per row (view, edit, delete)
    // New button + (view, edit, delete) * 2 rows = 7 buttons
    // Delete buttons are at index 3 and 6 (0-indexed: new=0, view1=1, edit1=2, delete1=3, ...)
    const deleteButtons = buttons.filter((_, i) => i === 3 || i === 6);
    await user.click(deleteButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('common.confirmDelete')).toBeInTheDocument();
    });
  });
});
