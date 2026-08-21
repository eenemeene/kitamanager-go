import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppSidebar } from '../app-sidebar';

jest.mock('next/navigation', () => ({
  usePathname: () => '/',
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

const mockToggleSidebar = jest.fn();
const mockSetMobileSidebarOpen = jest.fn();
let mockUiStore = {
  sidebarCollapsed: false,
  toggleSidebar: mockToggleSidebar,
  selectedOrganizationId: null as number | null,
  sidebarMobileOpen: false,
  setMobileSidebarOpen: mockSetMobileSidebarOpen,
};

jest.mock('@/stores/ui-store', () => ({
  useUiStore: () => mockUiStore,
}));

jest.mock('@/stores/auth-store', () => ({
  useAuthStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      user: { id: 1, is_superadmin: true },
      orgRoleMap: new Map([[1, 'admin']]),
    };
    return selector ? selector(state) : state;
  },
}));

jest.mock('../org-selector', () => ({
  OrgSelector: () => <div data-testid="org-selector">OrgSelector</div>,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: { getHealth: jest.fn().mockResolvedValue({ status: 'healthy', version: 'test123' }) },
}));

function renderWithQueryClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('AppSidebar', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUiStore = {
      sidebarCollapsed: false,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: null,
      sidebarMobileOpen: false,
      setMobileSidebarOpen: mockSetMobileSidebarOpen,
    };
  });

  it('renders main navigation links (Organizations, Government Fundings)', () => {
    renderWithQueryClient(<AppSidebar />);

    expect(screen.getByText('nav.organizations')).toBeInTheDocument();
    expect(screen.getByText('nav.governmentFundings')).toBeInTheDocument();
  });

  it('renders org selector', () => {
    renderWithQueryClient(<AppSidebar />);

    expect(screen.getByTestId('org-selector')).toBeInTheDocument();
  });

  it('hides org-scoped navigation when no org selected', () => {
    renderWithQueryClient(<AppSidebar />);

    expect(screen.queryByText('nav.users')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.employees')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.children')).not.toBeInTheDocument();
  });

  it('shows org-scoped navigation when org selected', () => {
    mockUiStore = {
      sidebarCollapsed: false,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: 1,
      sidebarMobileOpen: false,
      setMobileSidebarOpen: mockSetMobileSidebarOpen,
    };

    renderWithQueryClient(<AppSidebar />);

    // Group headers
    expect(screen.getByText('nav.groupDailyOperations')).toBeInTheDocument();
    expect(screen.getByText('nav.groupPeople')).toBeInTheDocument();
    expect(screen.getByText('nav.groupFinance')).toBeInTheDocument();
    expect(screen.getByText('nav.groupSettings')).toBeInTheDocument();

    // Daily Operations group
    expect(screen.getByText('nav.dashboard')).toBeInTheDocument();
    expect(screen.getByText('nav.attendance')).toBeInTheDocument();
    expect(screen.getByText('nav.sections')).toBeInTheDocument();

    // People group
    expect(screen.getByText('nav.children')).toBeInTheDocument();
    expect(screen.getByText('nav.employees')).toBeInTheDocument();

    // Finance group
    expect(screen.getByText('nav.governmentFundingBills')).toBeInTheDocument();
    expect(screen.getByText('nav.budgetItems')).toBeInTheDocument();
    expect(screen.getByText('nav.statistics')).toBeInTheDocument();

    // Settings group
    expect(screen.getByText('nav.payPlans')).toBeInTheDocument();
    expect(screen.getByText('nav.users')).toBeInTheDocument();
  });

  it('renders collapse/toggle sidebar button', () => {
    renderWithQueryClient(<AppSidebar />);

    const toggleButton = screen.getByLabelText('common.toggleSidebar');
    expect(toggleButton).toBeInTheDocument();
  });

  it('hides text labels when sidebar is collapsed', () => {
    mockUiStore = {
      sidebarCollapsed: true,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: null,
      sidebarMobileOpen: false,
      setMobileSidebarOpen: mockSetMobileSidebarOpen,
    };

    renderWithQueryClient(<AppSidebar />);

    // When collapsed, navigation text labels are hidden
    expect(screen.queryByText('nav.organizations')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.governmentFundings')).not.toBeInTheDocument();
    // Org selector is also hidden when collapsed
    expect(screen.queryByTestId('org-selector')).not.toBeInTheDocument();
  });
});
