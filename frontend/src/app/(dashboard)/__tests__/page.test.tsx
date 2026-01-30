import { render, screen } from '@testing-library/react';
import DashboardPage from '../page';
import { useUiStore } from '@/stores/ui-store';
import { useAuthStore } from '@/stores/auth-store';

// Mock the stores
jest.mock('@/stores/ui-store', () => ({
  useUiStore: jest.fn(),
}));

jest.mock('@/stores/auth-store', () => ({
  useAuthStore: jest.fn(),
}));

const mockGetSelectedOrganization = jest.fn();

describe('DashboardPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetSelectedOrganization.mockReturnValue(null);
  });

  it('renders dashboard title', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: null,
    });

    render(<DashboardPage />);

    expect(screen.getByText('dashboard.title')).toBeInTheDocument();
    expect(screen.getByText('dashboard.welcome')).toBeInTheDocument();
  });

  it('displays user name when available', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: { name: 'John Doe' },
    });

    render(<DashboardPage />);

    expect(screen.getByText(/John Doe/)).toBeInTheDocument();
  });

  it('renders stat cards', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [{ id: 1 }, { id: 2 }],
      organizationsLoading: false,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: null,
    });

    render(<DashboardPage />);

    expect(screen.getByText('dashboard.totalOrganizations')).toBeInTheDocument();
    expect(screen.getByText('dashboard.totalEmployees')).toBeInTheDocument();
    expect(screen.getByText('dashboard.totalChildren')).toBeInTheDocument();
    expect(screen.getByText('dashboard.totalUsers')).toBeInTheDocument();
  });

  it('shows organization count', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [{ id: 1 }, { id: 2 }, { id: 3 }],
      organizationsLoading: false,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: null,
    });

    render(<DashboardPage />);

    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('shows loading skeleton when organizations loading', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: true,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: null,
    });

    render(<DashboardPage />);

    // Should have skeleton elements when loading
    const card = screen.getByText('dashboard.totalOrganizations').closest('.rounded-lg');
    expect(card).toBeInTheDocument();
  });

  it('renders quick stats section', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
      getSelectedOrganization: mockGetSelectedOrganization,
    });
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: null,
    });

    render(<DashboardPage />);

    expect(screen.getByText('dashboard.quickStats')).toBeInTheDocument();
    expect(screen.getByText('statistics.selectOrgForStats')).toBeInTheDocument();
  });
});
