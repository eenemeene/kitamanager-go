import { create } from 'zustand';
import { apiClient } from '@/lib/api/client';
import type { User, LoginRequest } from '@/lib/api/types';
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

  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  loadUser: () => Promise<void>;
  checkAuth: () => boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  userLoading: false,
  userLoaded: false,
  isAuthenticated: false,
  hasHydrated: false,

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
  // Clear refresh token (refresh already failed or unavailable)
  apiClient.setRefreshToken(null);
  // Clear local state without calling logout endpoint (already unauthorized)
  if (typeof window !== 'undefined') {
    localStorage.removeItem('selectedOrgId');
  }
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    userLoaded: false,
  });
});
