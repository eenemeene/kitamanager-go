import { useUiStore } from '../ui-store';
import { apiClient } from '@/lib/api/client';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getOrganizations: jest.fn(),
  },
}));

const mockOrganizations = [
  {
    id: 1,
    name: 'Org 1',
    active: true,
    state: 'berlin',
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin@example.com',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Org 2',
    active: true,
    state: 'berlin',
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin@example.com',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 3,
    name: 'Org 3',
    active: false,
    state: 'berlin',
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin@example.com',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('useUiStore', () => {
  beforeEach(() => {
    // Reset store state
    useUiStore.setState({
      sidebarCollapsed: false,
      selectedOrganizationId: null,
      organizations: [],
      organizationsLoading: false,
    });
    jest.clearAllMocks();
  });

  describe('sidebar', () => {
    it('toggles sidebar collapsed state', () => {
      expect(useUiStore.getState().sidebarCollapsed).toBe(false);

      useUiStore.getState().toggleSidebar();
      expect(useUiStore.getState().sidebarCollapsed).toBe(true);

      useUiStore.getState().toggleSidebar();
      expect(useUiStore.getState().sidebarCollapsed).toBe(false);
    });

    it('sets sidebar collapsed directly', () => {
      useUiStore.getState().setSidebarCollapsed(true);
      expect(useUiStore.getState().sidebarCollapsed).toBe(true);

      useUiStore.getState().setSidebarCollapsed(false);
      expect(useUiStore.getState().sidebarCollapsed).toBe(false);
    });
  });

  describe('organization selection', () => {
    it('sets selected organization', () => {
      useUiStore.getState().setSelectedOrganization(1);
      expect(useUiStore.getState().selectedOrganizationId).toBe(1);

      useUiStore.getState().setSelectedOrganization(null);
      expect(useUiStore.getState().selectedOrganizationId).toBeNull();
    });

    it('syncs from route when different', () => {
      useUiStore.setState({ selectedOrganizationId: 1 });

      useUiStore.getState().syncFromRoute(2);
      expect(useUiStore.getState().selectedOrganizationId).toBe(2);
    });

    it('does not sync from route when same', () => {
      useUiStore.setState({ selectedOrganizationId: 1 });

      useUiStore.getState().syncFromRoute(1);
      expect(useUiStore.getState().selectedOrganizationId).toBe(1);
    });

    it('does not sync from route when null', () => {
      useUiStore.setState({ selectedOrganizationId: 1 });

      useUiStore.getState().syncFromRoute(null);
      expect(useUiStore.getState().selectedOrganizationId).toBe(1);
    });
  });

  describe('isValidOrganization', () => {
    it('returns true for valid organization', () => {
      useUiStore.setState({ organizations: mockOrganizations });

      expect(useUiStore.getState().isValidOrganization(1)).toBe(true);
      expect(useUiStore.getState().isValidOrganization(2)).toBe(true);
    });

    it('returns false for invalid organization', () => {
      useUiStore.setState({ organizations: mockOrganizations });

      expect(useUiStore.getState().isValidOrganization(999)).toBe(false);
    });
  });

  describe('getSelectedOrganization', () => {
    it('returns selected organization', () => {
      useUiStore.setState({
        organizations: mockOrganizations,
        selectedOrganizationId: 2,
      });

      const selected = useUiStore.getState().getSelectedOrganization();
      expect(selected).toEqual(mockOrganizations[1]);
    });

    it('returns null when no organization selected', () => {
      useUiStore.setState({
        organizations: mockOrganizations,
        selectedOrganizationId: null,
      });

      expect(useUiStore.getState().getSelectedOrganization()).toBeNull();
    });

    it('returns null when selected organization not found', () => {
      useUiStore.setState({
        organizations: mockOrganizations,
        selectedOrganizationId: 999,
      });

      expect(useUiStore.getState().getSelectedOrganization()).toBeNull();
    });
  });

  describe('fetchOrganizations', () => {
    it('fetches and stores organizations', async () => {
      (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockOrganizations);

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().organizations).toEqual(mockOrganizations);
      expect(useUiStore.getState().organizationsLoading).toBe(false);
    });

    it('auto-selects first org when none selected', async () => {
      (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockOrganizations);
      useUiStore.setState({ selectedOrganizationId: null });

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().selectedOrganizationId).toBe(1);
    });

    it('keeps selected org if still valid', async () => {
      (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockOrganizations);
      useUiStore.setState({ selectedOrganizationId: 2 });

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().selectedOrganizationId).toBe(2);
    });

    it('resets to first org if selected org no longer valid', async () => {
      (apiClient.getOrganizations as jest.Mock).mockResolvedValue(mockOrganizations);
      useUiStore.setState({ selectedOrganizationId: 999 });

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().selectedOrganizationId).toBe(1);
    });

    it('sets selectedOrganizationId to null when no organizations', async () => {
      (apiClient.getOrganizations as jest.Mock).mockResolvedValue([]);
      useUiStore.setState({ selectedOrganizationId: 1 });

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().selectedOrganizationId).toBeNull();
    });

    it('handles API error gracefully', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      (apiClient.getOrganizations as jest.Mock).mockRejectedValue(new Error('Network error'));

      await useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().organizationsLoading).toBe(false);
      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('sets loading state during fetch', async () => {
      let resolvePromise: (value: unknown) => void;
      const promise = new Promise((resolve) => {
        resolvePromise = resolve;
      });
      (apiClient.getOrganizations as jest.Mock).mockReturnValue(promise);

      const fetchPromise = useUiStore.getState().fetchOrganizations();

      expect(useUiStore.getState().organizationsLoading).toBe(true);

      resolvePromise!(mockOrganizations);
      await fetchPromise;

      expect(useUiStore.getState().organizationsLoading).toBe(false);
    });
  });
});
