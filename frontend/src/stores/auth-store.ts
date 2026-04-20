import { create } from 'zustand';
import { apiClient } from '@/lib/api/client';
import type { User, LoginRequest, Role, UserMembership } from '@/lib/api/types';
import { getCookie } from '@/lib/utils';

/**
 * Check if CSRF cookie is present (indicates authenticated session).
 * The access_token is HttpOnly so we can't read it from JS,
 * but the csrf_token is JS-readable and set alongside it.
 */
function hasAuthCookie(): boolean {
  return getCookie('csrf_token') !== null;
}

interface AuthState {
  user: Partial<User> | null;
  userLoading: boolean;
  userLoaded: boolean;
  isAuthenticated: boolean;
  hasHydrated: boolean;
  memberships: UserMembership[];
  orgRoleMap: Map<number, Role>;

  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  loadUser: () => Promise<void>;
  checkAuth: () => boolean;
  setHasHydrated: (state: boolean) => void;
}

function buildOrgRoleMap(memberships: UserMembership[]): Map<number, Role> {
  const map = new Map<number, Role>();
  for (const m of memberships) {
    if (m.organization_id) {
      map.set(m.organization_id, m.role);
    }
  }
  return map;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  userLoading: false,
  userLoaded: false,
  isAuthenticated: false,
  hasHydrated: false,
  memberships: [],
  orgRoleMap: new Map(),

  setHasHydrated: (state: boolean) => {
    set({ hasHydrated: state });
  },

  login: async (credentials: LoginRequest) => {
    // Login sets HttpOnly cookies automatically
    await apiClient.login(credentials);

    set({ isAuthenticated: true });

    // Fetch full user data using the new /me endpoint
    try {
      const userData = await apiClient.getCurrentUser();
      set({ user: userData, userLoaded: true });

      // Fetch memberships for role-based navigation
      if (userData.id) {
        try {
          const { memberships } = await apiClient.getUserMemberships(userData.id);
          set({ memberships, orgRoleMap: buildOrgRoleMap(memberships) });
        } catch {
          // Non-critical: navigation will show all items if memberships fail
        }
      }
    } catch {
      set({ userLoaded: true });
    }
  },

  logout: async () => {
    try {
      await apiClient.logout();
    } catch {
      // Ignore logout errors - cookies may already be cleared
    }
    if (typeof window !== 'undefined') {
      localStorage.removeItem('selectedOrgId');
    }
    set({
      user: null,
      isAuthenticated: false,
      userLoaded: false,
      memberships: [],
      orgRoleMap: new Map(),
    });
  },

  loadUser: async () => {
    if (!hasAuthCookie()) {
      set({ userLoaded: true, isAuthenticated: false });
      return;
    }

    set({ userLoading: true, isAuthenticated: true });
    try {
      // Try to get current user info - backend will use the cookie
      const userData = await apiClient.getCurrentUser();
      set({ user: userData, userLoaded: true, userLoading: false });

      // Fetch memberships for role-based navigation
      if (userData.id) {
        try {
          const { memberships } = await apiClient.getUserMemberships(userData.id);
          set({ memberships, orgRoleMap: buildOrgRoleMap(memberships) });
        } catch {
          // Non-critical: navigation will show all items if memberships fail
        }
      }
    } catch {
      // Cookie may be expired or invalid
      set({
        user: null,
        userLoaded: true,
        userLoading: false,
        isAuthenticated: false,
      });
    }
  },

  checkAuth: () => {
    const authenticated = hasAuthCookie();
    set({ isAuthenticated: authenticated });
    return authenticated;
  },
}));

// Initialize auth state on load
if (typeof window !== 'undefined') {
  // Check auth status immediately
  const store = useAuthStore.getState();
  store.setHasHydrated(true);
  if (store.checkAuth()) {
    store.loadUser();
  }
}

// Set up unauthorized callback
apiClient.setOnUnauthorized(() => {
  // Clear session flag (refresh already failed or unavailable)
  apiClient.setHasSession(false);
  // Clear local state without calling logout endpoint (already unauthorized)
  if (typeof window !== 'undefined') {
    localStorage.removeItem('selectedOrgId');
    // Clear the JS-readable csrf_token cookie. Without this, hasAuthCookie()
    // keeps returning true after the server rejected our tokens, which causes
    // components to re-fire authenticated requests in an infinite loop. The
    // HttpOnly access/refresh cookies are cleared by the server on 401.
    document.cookie = 'csrf_token=; path=/; max-age=0; SameSite=Strict';
  }
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    userLoaded: false,
    memberships: [],
    orgRoleMap: new Map(),
  });
});
