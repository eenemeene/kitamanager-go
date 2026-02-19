import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { apiClient } from '@/lib/api/client';
import type { Organization } from '@/lib/api/types';

interface UiState {
  sidebarCollapsed: boolean;
  sidebarMobileOpen: boolean;
  selectedOrganizationId: number | null;
  organizations: Organization[];
  organizationsLoading: boolean;

  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleMobileSidebar: () => void;
  setMobileSidebarOpen: (open: boolean) => void;
  setSelectedOrganization: (orgId: number | null) => void;
  syncFromRoute: (orgId: number | null) => void;
  isValidOrganization: (orgId: number) => boolean;
  fetchOrganizations: () => Promise<void>;
  getSelectedOrganization: () => Organization | null;
}

export const useUiStore = create<UiState>()(
  persist(
    (set, get) => ({
      sidebarCollapsed: false,
      sidebarMobileOpen: false,
      selectedOrganizationId: null,
      organizations: [],
      organizationsLoading: false,

      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      toggleMobileSidebar: () => {
        set((state) => ({ sidebarMobileOpen: !state.sidebarMobileOpen }));
      },

      setMobileSidebarOpen: (open: boolean) => {
        set({ sidebarMobileOpen: open });
      },

      setSelectedOrganization: (orgId: number | null) => {
        set({ selectedOrganizationId: orgId });
      },

      syncFromRoute: (orgId: number | null) => {
        const { selectedOrganizationId } = get();
        if (orgId && orgId !== selectedOrganizationId) {
          set({ selectedOrganizationId: orgId });
        }
      },

      isValidOrganization: (orgId: number) => {
        return get().organizations.some((o) => o.id === orgId);
      },

      fetchOrganizations: async () => {
        set({ organizationsLoading: true });
        try {
          const organizations = await apiClient.getOrganizationsAll();
          const { selectedOrganizationId } = get();

          // Auto-select first org if none selected and orgs exist
          let newSelectedId = selectedOrganizationId;
          if (!selectedOrganizationId && organizations.length > 0) {
            newSelectedId = organizations[0].id;
          } else if (
            selectedOrganizationId &&
            !organizations.some((o) => o.id === selectedOrganizationId)
          ) {
            // Selected org no longer valid, reset
            newSelectedId = organizations.length > 0 ? organizations[0].id : null;
          }

          set({
            organizations,
            selectedOrganizationId: newSelectedId,
            organizationsLoading: false,
          });
        } catch (error) {
          console.error('Failed to load organizations:', error);
          set({ organizationsLoading: false });
        }
      },

      getSelectedOrganization: () => {
        const { organizations, selectedOrganizationId } = get();
        return organizations.find((o) => o.id === selectedOrganizationId) || null;
      },
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        selectedOrganizationId: state.selectedOrganizationId,
      }),
    }
  )
);
