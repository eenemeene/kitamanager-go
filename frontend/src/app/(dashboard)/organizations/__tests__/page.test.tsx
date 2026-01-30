import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import OrganizationsPage from '../page';
import { apiClient } from '@/lib/api/client';

// Mock API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getOrganizations: jest.fn(),
    createOrganization: jest.fn(),
    updateOrganization: jest.fn(),
    deleteOrganization: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

// Mock toast
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({
    toast: jest.fn(),
  }),
}));

const mockOrganizations = [
  { id: 1, name: 'Kita Sonnenschein', state: 'berlin', active: true },
  { id: 2, name: 'Kita Regenbogen', state: 'berlin', active: false },
];

const mockPaginatedResponse = {
  data: mockOrganizations,
  total: 2,
  page: 1,
  limit: 30,
  total_pages: 1,
};

const mockEmptyResponse = {
  data: [],
  total: 0,
  page: 1,
  limit: 30,
  total_pages: 0,
};

function renderWithProviders(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('OrganizationsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<OrganizationsPage />);

    // Title appears in both the page header and card header
    const titles = screen.getAllByText('organizations.title');
    expect(titles.length).toBeGreaterThanOrEqual(1);
  });

  it('renders new organization button', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<OrganizationsPage />);

    expect(screen.getByText('organizations.newOrganization')).toBeInTheDocument();
  });

  it('shows loading skeleton while fetching', async () => {
    (apiClient.getOrganizations as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderWithProviders(<OrganizationsPage />);

    // Look for skeleton elements (they have specific classes)
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('displays organizations in table', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('Kita Sonnenschein')).toBeInTheDocument();
    });

    expect(screen.getByText('Kita Regenbogen')).toBeInTheDocument();
  });

  it('displays active badge for active organizations', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('Kita Sonnenschein')).toBeInTheDocument();
    });

    // Should have both active and inactive badges
    expect(screen.getByText('common.active')).toBeInTheDocument();
    expect(screen.getByText('common.inactive')).toBeInTheDocument();
  });

  it('shows no results message when empty', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });
  });

  it('renders table headers', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });

    expect(screen.getByText('common.id')).toBeInTheDocument();
    expect(screen.getByText('common.name')).toBeInTheDocument();
    expect(screen.getByText('states.state')).toBeInTheDocument();
    expect(screen.getByText('common.status')).toBeInTheDocument();
    expect(screen.getByText('common.actions')).toBeInTheDocument();
  });

  it('has new organization button that is clickable', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });

    const newButton = screen.getByText('organizations.newOrganization');
    expect(newButton).toBeInTheDocument();
    expect(newButton.closest('button')).not.toBeDisabled();
  });

  it('renders edit and delete buttons for each organization', async () => {
    (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<OrganizationsPage />);

    await waitFor(() => {
      expect(screen.getByText('Kita Sonnenschein')).toBeInTheDocument();
    });

    // Should have 2 edit buttons and 2 delete buttons (one for each org)
    const buttons = screen.getAllByRole('button');
    // New org button + 2 edit + 2 delete = 5 buttons minimum
    expect(buttons.length).toBeGreaterThanOrEqual(5);
  });
});
