import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppSidebar } from '../app-sidebar';

let mockPathname = '/';
jest.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
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

// ---------------------------------------------------------------------------
// Which item is lit
// ---------------------------------------------------------------------------
//
// Both of these were prefix matches that did not stop at a path segment, so
// two items claimed the same page.
describe('AppSidebar active item', () => {
  // `classList`, not a substring of `className`: the expanded parent of a
  // submenu carries `bg-sidebar-active/10`, which contains the active class as
  // a substring while meaning something else.
  const isLit = (el: Element) => el.classList.contains('bg-sidebar-active');

  function linkFor(name: string) {
    return screen.getAllByRole('link', { name }).find((el) => el.getAttribute('href') !== '#')!;
  }

  beforeEach(() => {
    jest.clearAllMocks();
    mockUiStore = {
      sidebarCollapsed: false,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: 1,
      sidebarMobileOpen: false,
      setMobileSidebarOpen: mockSetMobileSidebarOpen,
    };
  });

  afterEach(() => {
    mockPathname = '/';
  });

  it('does not light Organizations up for every org-scoped page', () => {
    // `/organizations` is a prefix of every route below it, so a superadmin saw
    // it active on top of Children for the whole session.
    mockPathname = '/organizations/1/children';
    renderWithQueryClient(<AppSidebar />);

    expect(isLit(linkFor('nav.organizations'))).toBe(false);
    expect(isLit(linkFor('nav.children'))).toBe(true);
  });

  it('lights Organizations up on the organizations list itself', () => {
    mockPathname = '/organizations';
    renderWithQueryClient(<AppSidebar />);

    expect(isLit(linkFor('nav.organizations'))).toBe(true);
  });

  it('does not light Statistics up on the Forecast page', () => {
    // Only reachable with the rail collapsed: expanded, Statistics renders as a
    // submenu parent and is styled from `anyChildActive` already. Collapsed it
    // falls through to the leaf branch, which matched its own href as a bare
    // substring -- and `/statistics` is a substring of `/statistics/forecast`,
    // a sibling item rather than one of its children. Both icons lit up.
    mockUiStore.sidebarCollapsed = true;
    mockPathname = '/organizations/1/statistics/forecast';
    renderWithQueryClient(<AppSidebar />);

    expect(isLit(linkFor('nav.statisticsForecast'))).toBe(true);
    expect(isLit(linkFor('nav.statistics'))).toBe(false);
  });

  it('still lights Statistics up on one of its own sub-pages', () => {
    mockUiStore.sidebarCollapsed = true;
    mockPathname = '/organizations/1/statistics/financials';
    renderWithQueryClient(<AppSidebar />);

    expect(isLit(linkFor('nav.statistics'))).toBe(true);
    expect(isLit(linkFor('nav.statisticsForecast'))).toBe(false);
  });

  it('lights a leaf up on its own detail pages', () => {
    mockPathname = '/organizations/1/children/7/contracts';
    renderWithQueryClient(<AppSidebar />);

    expect(isLit(linkFor('nav.children'))).toBe(true);
  });
});
