import { useAuthStore } from '../auth-store';
import { apiClient } from '@/lib/api/client';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    login: jest.fn(),
    logout: jest.fn(),
    getCurrentUser: jest.fn(),
    setOnUnauthorized: jest.fn(),
  },
}));

// Mock document.cookie for cookie-based auth
let mockCookies: Record<string, string> = {};

Object.defineProperty(document, 'cookie', {
  get: () => {
    return Object.entries(mockCookies)
      .map(([key, value]) => `${key}=${value}`)
      .join('; ');
  },
  set: (value: string) => {
    // Parse cookie string like "name=value; path=/; max-age=3600"
    const parts = value.split(';').map((p) => p.trim());
    const [nameValue] = parts;
    const [name, val] = nameValue.split('=');
    if (parts.some((p) => p.startsWith('max-age=-'))) {
      // Cookie deletion
      delete mockCookies[name];
    } else {
      mockCookies[name] = val;
    }
  },
});

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('useAuthStore', () => {
  beforeEach(() => {
    // Reset store state
    useAuthStore.setState({
      user: null,
      userLoading: false,
      userLoaded: false,
      isAuthenticated: false,
      hasHydrated: false,
    });
    mockCookies = {};
    localStorageMock.clear();
    jest.clearAllMocks();
  });

  describe('login', () => {
    it('calls login API and fetches user data on success', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({
        token: 'mock-token',
        refresh_token: 'mock-refresh',
      });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
      });

      await useAuthStore.getState().login({ email: 'test@example.com', password: 'password' });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
      });
      expect(apiClient.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password',
      });
      expect(apiClient.getCurrentUser).toHaveBeenCalled();
    });

    it('sets userLoaded even if getCurrentUser fails', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({
        token: 'mock-token',
      });
      (apiClient.getCurrentUser as jest.Mock).mockRejectedValue(new Error('Network error'));

      await useAuthStore.getState().login({ email: 'test@example.com', password: 'password' });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.userLoaded).toBe(true);
    });

    it('handles login failure', async () => {
      (apiClient.login as jest.Mock).mockRejectedValue(new Error('Invalid credentials'));

      await expect(
        useAuthStore.getState().login({ email: 'test@example.com', password: 'wrong' })
      ).rejects.toThrow('Invalid credentials');

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('logout', () => {
    it('calls logout API and clears state', async () => {
      (apiClient.logout as jest.Mock).mockResolvedValue(undefined);

      useAuthStore.setState({
        user: { id: 1, email: 'test@example.com' },
        isAuthenticated: true,
        userLoaded: true,
      });
      localStorage.setItem('selectedOrgId', '1');

      await useAuthStore.getState().logout();

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.userLoaded).toBe(false);
      expect(localStorage.getItem('selectedOrgId')).toBeNull();
      expect(apiClient.logout).toHaveBeenCalled();
    });

    it('clears state even if logout API fails', async () => {
      (apiClient.logout as jest.Mock).mockRejectedValue(new Error('Network error'));

      useAuthStore.setState({
        user: { id: 1, email: 'test@example.com' },
        isAuthenticated: true,
      });

      await useAuthStore.getState().logout();

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('checkAuth', () => {
    it('returns true when csrf_token cookie is present', () => {
      mockCookies['csrf_token'] = 'test-csrf-token';

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(true);
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('returns false when no csrf_token cookie', () => {
      mockCookies = {};

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(false);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('returns false when only other cookies are present', () => {
      mockCookies['other_cookie'] = 'some-value';

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(false);
    });
  });

  describe('loadUser', () => {
    it('loads user data when csrf_token cookie is present', async () => {
      mockCookies['csrf_token'] = 'test-csrf-token';

      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        is_superadmin: true,
      });

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.user).toEqual({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        is_superadmin: true,
      });
      expect(state.userLoaded).toBe(true);
      expect(state.userLoading).toBe(false);
      expect(state.isAuthenticated).toBe(true);
    });

    it('sets userLoaded and clears auth on API error', async () => {
      mockCookies['csrf_token'] = 'test-csrf-token';

      (apiClient.getCurrentUser as jest.Mock).mockRejectedValue(new Error('Unauthorized'));

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.userLoaded).toBe(true);
      expect(state.isAuthenticated).toBe(false);
    });

    it('sets userLoaded true when no auth cookie', async () => {
      mockCookies = {};

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.userLoaded).toBe(true);
      expect(state.isAuthenticated).toBe(false);
      expect(apiClient.getCurrentUser).not.toHaveBeenCalled();
    });
  });

  describe('no localStorage token usage', () => {
    it('should not store token in localStorage on login', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ token: 'mock-token' });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 1, email: 'test@test.com' });

      await useAuthStore.getState().login({ email: 'test@test.com', password: 'password' });

      expect(localStorage.getItem('token')).toBeNull();
    });

    it('should not read token from localStorage', () => {
      localStorage.setItem('token', 'some-old-token');

      // Reset and check auth - should not use localStorage token
      useAuthStore.setState({ isAuthenticated: false });
      const result = useAuthStore.getState().checkAuth();

      // Without csrf_token cookie, should not be authenticated
      expect(result).toBe(false);
    });
  });
});
