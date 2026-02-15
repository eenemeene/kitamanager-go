import { render, screen } from '@testing-library/react';
import { AppSidebar } from '../app-sidebar';

jest.mock('next/navigation', () => ({
  usePathname: () => '/',
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

const mockToggleSidebar = jest.fn();
let mockUiStore = {
  sidebarCollapsed: false,
  toggleSidebar: mockToggleSidebar,
  selectedOrganizationId: null as number | null,
};

jest.mock('@/stores/ui-store', () => ({
  useUiStore: () => mockUiStore,
}));

jest.mock('../org-selector', () => ({
  OrgSelector: () => <div data-testid="org-selector">OrgSelector</div>,
}));

describe('AppSidebar', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUiStore = {
      sidebarCollapsed: false,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: null,
    };
  });

  it('renders main navigation links (Organizations, Government Fundings)', () => {
    render(<AppSidebar />);

    expect(screen.getByText('nav.organizations')).toBeInTheDocument();
    expect(screen.getByText('nav.governmentFundings')).toBeInTheDocument();
  });

  it('renders org selector', () => {
    render(<AppSidebar />);

    expect(screen.getByTestId('org-selector')).toBeInTheDocument();
  });

  it('hides org-scoped navigation when no org selected', () => {
    render(<AppSidebar />);

    expect(screen.queryByText('nav.users')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.employees')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.children')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.groups')).not.toBeInTheDocument();
  });

  it('shows org-scoped navigation when org selected', () => {
    mockUiStore = {
      sidebarCollapsed: false,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: 1,
    };

    render(<AppSidebar />);

    expect(screen.getByText('nav.dashboard')).toBeInTheDocument();
    expect(screen.getByText('nav.users')).toBeInTheDocument();
    expect(screen.getByText('nav.employees')).toBeInTheDocument();
    expect(screen.getByText('nav.children')).toBeInTheDocument();
    expect(screen.getByText('nav.groups')).toBeInTheDocument();
    expect(screen.getByText('nav.sections')).toBeInTheDocument();
    expect(screen.getByText('nav.statistics')).toBeInTheDocument();
    // Pay Plans is nested under Employees (collapsed by default)
    expect(screen.queryByText('nav.payPlans')).not.toBeInTheDocument();
  });

  it('renders collapse/toggle sidebar button', () => {
    render(<AppSidebar />);

    const toggleButton = screen.getByLabelText('common.toggleSidebar');
    expect(toggleButton).toBeInTheDocument();
  });

  it('hides text labels when sidebar is collapsed', () => {
    mockUiStore = {
      sidebarCollapsed: true,
      toggleSidebar: mockToggleSidebar,
      selectedOrganizationId: null,
    };

    render(<AppSidebar />);

    // When collapsed, navigation text labels are hidden
    expect(screen.queryByText('nav.organizations')).not.toBeInTheDocument();
    expect(screen.queryByText('nav.governmentFundings')).not.toBeInTheDocument();
    // Org selector is also hidden when collapsed
    expect(screen.queryByTestId('org-selector')).not.toBeInTheDocument();
  });
});
