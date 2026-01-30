import { useAuthStore } from '../auth-store';
import { apiClient } from '@/lib/api/client';

// Mock the API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    login: jest.fn(),
    getUser: jest.fn(),
    setOnUnauthorized: jest.fn(),
  },
}));

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

// Helper to create a valid JWT token
function createMockToken(payload: { user_id: number; email: string; exp: number }): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const body = btoa(JSON.stringify(payload));
  const signature = 'mock-signature';
  return `${header}.${body}.${signature}`;
}

describe('useAuthStore', () => {
  beforeEach(() => {
    // Reset store state
    useAuthStore.setState({
      token: null,
      user: null,
      userLoading: false,
      userLoaded: false,
      isAuthenticated: false,
    });
    localStorageMock.clear();
    jest.clearAllMocks();
  });

  describe('login', () => {
    it('sets token and user on successful login', async () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
      });

      (apiClient.login as jest.Mock).mockResolvedValue({ token: mockToken });
      (apiClient.getUser as jest.Mock).mockResolvedValue({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
      });

      await useAuthStore.getState().login({ email: 'test@example.com', password: 'password' });

      const state = useAuthStore.getState();
      expect(state.token).toBe(mockToken);
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
      });
      expect(localStorage.getItem('token')).toBe(mockToken);
    });

    it('handles login failure', async () => {
      (apiClient.login as jest.Mock).mockRejectedValue(new Error('Invalid credentials'));

      await expect(
        useAuthStore.getState().login({ email: 'test@example.com', password: 'wrong' })
      ).rejects.toThrow('Invalid credentials');

      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('logout', () => {
    it('clears token, user, and localStorage', () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
      });

      useAuthStore.setState({
        token: mockToken,
        user: { id: 1, email: 'test@example.com' },
        isAuthenticated: true,
      });
      localStorage.setItem('token', mockToken);
      localStorage.setItem('selectedOrgId', '1');

      useAuthStore.getState().logout();

      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(localStorage.getItem('token')).toBeNull();
      expect(localStorage.getItem('selectedOrgId')).toBeNull();
    });
  });

  describe('setToken', () => {
    it('sets token and updates isAuthenticated', () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
      });

      useAuthStore.getState().setToken(mockToken);

      const state = useAuthStore.getState();
      expect(state.token).toBe(mockToken);
      expect(state.isAuthenticated).toBe(true);
      expect(localStorage.getItem('token')).toBe(mockToken);
    });

    it('clears token when set to null', () => {
      localStorage.setItem('token', 'some-token');
      useAuthStore.setState({ token: 'some-token', isAuthenticated: true });

      useAuthStore.getState().setToken(null);

      const state = useAuthStore.getState();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(localStorage.getItem('token')).toBeNull();
    });
  });

  describe('checkAuth', () => {
    it('returns true for valid non-expired token', () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600, // 1 hour from now
      });

      useAuthStore.setState({ token: mockToken });

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(true);
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('returns false and logs out for expired token', () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) - 3600, // 1 hour ago
      });

      useAuthStore.setState({ token: mockToken, isAuthenticated: true });

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(false);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().token).toBeNull();
    });

    it('returns false when no token', () => {
      useAuthStore.setState({ token: null });

      const result = useAuthStore.getState().checkAuth();

      expect(result).toBe(false);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });
  });

  describe('loadUser', () => {
    it('loads user data from API', async () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
      });

      (apiClient.getUser as jest.Mock).mockResolvedValue({
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        is_superadmin: true,
      });

      useAuthStore.setState({ token: mockToken });

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
    });

    it('falls back to token data on API error', async () => {
      const mockToken = createMockToken({
        user_id: 1,
        email: 'test@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
      });

      (apiClient.getUser as jest.Mock).mockRejectedValue(new Error('Network error'));

      useAuthStore.setState({ token: mockToken });

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.user).toEqual({
        id: 1,
        email: 'test@example.com',
      });
      expect(state.userLoaded).toBe(true);
    });

    it('sets userLoaded true when no token', async () => {
      useAuthStore.setState({ token: null });

      await useAuthStore.getState().loadUser();

      expect(useAuthStore.getState().userLoaded).toBe(true);
    });
  });
});
