import { render, screen, fireEvent } from '@testing-library/react';
import { OrgSelector } from '../org-selector';
import { useUiStore } from '@/stores/ui-store';
import { useAuthStore } from '@/stores/auth-store';

// Mock the stores
jest.mock('@/stores/ui-store', () => ({
  useUiStore: jest.fn(),
}));

jest.mock('@/stores/auth-store', () => ({
  useAuthStore: jest.fn(),
}));

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

const mockOrganizations = [
  { id: 1, name: 'Kita Sonnenschein', active: true, state: 'berlin' },
  { id: 2, name: 'Kita Regenbogen', active: true, state: 'berlin' },
];

describe('OrgSelector', () => {
  const mockSetSelectedOrganization = jest.fn();
  const mockFetchOrganizations = jest.fn();
  const mockGetSelectedOrganization = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockGetSelectedOrganization.mockReturnValue(null);
  });

  it('shows loading state when organizations are loading', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: true,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    expect(screen.getByText('common.loading')).toBeInTheDocument();
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('shows select org prompt when no org selected', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: mockOrganizations,
      organizationsLoading: false,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    expect(screen.getByText('organizations.selectOrg')).toBeInTheDocument();
  });

  it('displays selected organization name', () => {
    mockGetSelectedOrganization.mockReturnValue(mockOrganizations[0]);
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: mockOrganizations,
      organizationsLoading: false,
      selectedOrganizationId: 1,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    expect(screen.getByText('Kita Sonnenschein')).toBeInTheDocument();
  });

  it('renders dropdown trigger button', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: mockOrganizations,
      organizationsLoading: false,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-haspopup', 'menu');
    expect(button).toHaveAttribute('aria-expanded', 'false');
  });

  it('has correct initial state for dropdown trigger', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: mockOrganizations,
      organizationsLoading: false,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('data-state', 'closed');
  });

  it('fetches organizations when authenticated and list is empty', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    expect(mockFetchOrganizations).toHaveBeenCalled();
  });

  it('renders with empty organizations list', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      isAuthenticated: true,
    });
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
      setSelectedOrganization: mockSetSelectedOrganization,
      fetchOrganizations: mockFetchOrganizations,
      getSelectedOrganization: mockGetSelectedOrganization,
    });

    render(<OrgSelector />);

    // Should show "select org" prompt when no organizations and none selected
    expect(screen.getByText('organizations.selectOrg')).toBeInTheDocument();
  });
});
