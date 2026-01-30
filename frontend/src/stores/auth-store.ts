import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { apiClient } from '@/lib/api/client';
import type { User, LoginRequest } from '@/lib/api/types';

interface JwtPayload {
  user_id: number;
  email: string;
  exp: number;
}

function parseJwt(token: string): JwtPayload | null {
  try {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
    return JSON.parse(jsonPayload);
  } catch {
    return null;
  }
}

interface AuthState {
  token: string | null;
  user: Partial<User> | null;
  userLoading: boolean;
  userLoaded: boolean;
  isAuthenticated: boolean;
  hasHydrated: boolean;

  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  setToken: (token: string | null) => void;
  loadUser: () => Promise<void>;
  checkAuth: () => boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      userLoading: false,
      userLoaded: false,
      isAuthenticated: false,
      hasHydrated: false,

      setHasHydrated: (state: boolean) => {
        set({ hasHydrated: state });
      },

      login: async (credentials: LoginRequest) => {
        const response = await apiClient.login(credentials);
        const token = response.token;

        // Store token in localStorage for API client
        if (typeof window !== 'undefined') {
          localStorage.setItem('token', token);
        }

        set({ token, isAuthenticated: true });

        // Parse user info and fetch full user data
        const payload = parseJwt(token);
        if (payload) {
          set({ user: { id: payload.user_id, email: payload.email } });
          try {
            const userData = await apiClient.getUser(payload.user_id);
            set({ user: userData, userLoaded: true });
          } catch {
            set({ userLoaded: true });
          }
        }
      },

      logout: () => {
        if (typeof window !== 'undefined') {
          localStorage.removeItem('token');
          localStorage.removeItem('selectedOrgId');
        }
        set({
          token: null,
          user: null,
          isAuthenticated: false,
          userLoaded: false,
        });
      },

      setToken: (token: string | null) => {
        if (typeof window !== 'undefined') {
          if (token) {
            localStorage.setItem('token', token);
          } else {
            localStorage.removeItem('token');
          }
        }
        set({ token, isAuthenticated: !!token });
      },

      loadUser: async () => {
        const { token } = get();
        if (!token) {
          set({ userLoaded: true });
          return;
        }

        const payload = parseJwt(token);
        if (!payload || payload.exp * 1000 <= Date.now()) {
          // Token expired
          get().logout();
          return;
        }

        set({ userLoading: true });
        try {
          const userData = await apiClient.getUser(payload.user_id);
          set({ user: userData, userLoaded: true, userLoading: false });
        } catch {
          // Keep basic info from token on error
          set({
            user: { id: payload.user_id, email: payload.email },
            userLoaded: true,
            userLoading: false,
          });
        }
      },

      checkAuth: () => {
        const { token } = get();
        if (!token) {
          set({ isAuthenticated: false });
          return false;
        }

        const payload = parseJwt(token);
        if (!payload || payload.exp * 1000 <= Date.now()) {
          get().logout();
          return false;
        }

        set({ isAuthenticated: true });
        return true;
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          // Mark hydration as complete
          state.setHasHydrated(true);

          if (state.token) {
            // Sync token to localStorage for API client
            if (typeof window !== 'undefined') {
              localStorage.setItem('token', state.token);
            }
            state.checkAuth();
            state.loadUser();
          }
        }
      },
    }
  )
);

// Set up unauthorized callback
apiClient.setOnUnauthorized(() => {
  useAuthStore.getState().logout();
});
