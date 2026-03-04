import { useAuthStore } from '../auth-store';
import { apiClient } from '@/lib/api/client';

// Mock the API client.
// The unauthorized callback is captured via globalThis because jest.mock is hoisted
// above all other statements, including variable declarations.
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    login: jest.fn(),
    logout: jest.fn(),
    getCurrentUser: jest.fn(),
    getUserMemberships: jest.fn(),
    setOnUnauthorized: jest.fn((cb: () => void) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (globalThis as any).__authTestUnauthorizedCb = cb;
    }),
    setHasSession: jest.fn(),
  },
}));

function getUnauthorizedCallback(): () => void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (globalThis as any).__authTestUnauthorizedCb;
}

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
      memberships: [],
      orgRoleMap: new Map(),
    });
    mockCookies = {};
    localStorageMock.clear();
    jest.clearAllMocks();
  });

  describe('login', () => {
    it('calls login API and fetches user data on success', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({
        expires_in: 3600,
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
        expires_in: 3600,
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

  describe('buildOrgRoleMap (via login/loadUser)', () => {
    it('produces an empty map from an empty memberships array', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 1, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({ memberships: [] });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      const state = useAuthStore.getState();
      expect(state.memberships).toEqual([]);
      expect(state.orgRoleMap.size).toBe(0);
    });

    it('maps organization_id to role for each membership', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 5, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
        memberships: [
          { user_id: 5, organization_id: 10, role: 'admin' },
          { user_id: 5, organization_id: 20, role: 'manager' },
          { user_id: 5, organization_id: 30, role: 'member' },
        ],
      });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      const { orgRoleMap, memberships } = useAuthStore.getState();
      expect(memberships).toHaveLength(3);
      expect(orgRoleMap.get(10)).toBe('admin');
      expect(orgRoleMap.get(20)).toBe('manager');
      expect(orgRoleMap.get(30)).toBe('member');
      expect(orgRoleMap.size).toBe(3);
    });

    it('skips memberships without organization_id', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 1, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
        memberships: [
          { user_id: 1, organization_id: 0, role: 'admin' },
          { user_id: 1, organization_id: null, role: 'manager' },
          { user_id: 1, organization_id: 42, role: 'member' },
        ],
      });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      const { orgRoleMap } = useAuthStore.getState();
      // organization_id 0 is falsy, so it should be skipped
      // organization_id null is falsy, so it should be skipped
      expect(orgRoleMap.size).toBe(1);
      expect(orgRoleMap.get(42)).toBe('member');
    });

    it('last membership wins when same org_id appears multiple times', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 1, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
        memberships: [
          { user_id: 1, organization_id: 10, role: 'member' },
          { user_id: 1, organization_id: 10, role: 'admin' },
        ],
      });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      const { orgRoleMap } = useAuthStore.getState();
      expect(orgRoleMap.size).toBe(1);
      expect(orgRoleMap.get(10)).toBe('admin');
    });
  });

  describe('login with memberships', () => {
    it('fetches memberships after successful user load', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 7, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
        memberships: [{ user_id: 7, organization_id: 3, role: 'admin' }],
      });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      expect(apiClient.getUserMemberships).toHaveBeenCalledWith(7);
      const state = useAuthStore.getState();
      expect(state.memberships).toHaveLength(1);
      expect(state.orgRoleMap.get(3)).toBe('admin');
    });

    it('completes login successfully even when memberships fetch fails', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 7, email: 'a@b.com' });
      (apiClient.getUserMemberships as jest.Mock).mockRejectedValue(new Error('500 Server Error'));

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual({ id: 7, email: 'a@b.com' });
      expect(state.userLoaded).toBe(true);
      // Memberships remain empty on failure
      expect(state.memberships).toEqual([]);
      expect(state.orgRoleMap.size).toBe(0);
    });

    it('does not fetch memberships when user data has no id', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ email: 'a@b.com' });

      await useAuthStore.getState().login({ email: 'a@b.com', password: 'pass' });

      expect(apiClient.getUserMemberships).not.toHaveBeenCalled();
      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.userLoaded).toBe(true);
    });
  });

  describe('loadUser with memberships', () => {
    it('fetches memberships after loading user', async () => {
      mockCookies['csrf_token'] = 'tok';
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 3, email: 'u@x.com' });
      (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
        memberships: [{ user_id: 3, organization_id: 5, role: 'manager' }],
      });

      await useAuthStore.getState().loadUser();

      expect(apiClient.getUserMemberships).toHaveBeenCalledWith(3);
      const state = useAuthStore.getState();
      expect(state.memberships).toHaveLength(1);
      expect(state.orgRoleMap.get(5)).toBe('manager');
      expect(state.userLoaded).toBe(true);
      expect(state.userLoading).toBe(false);
    });

    it('completes loadUser successfully even when memberships fetch fails', async () => {
      mockCookies['csrf_token'] = 'tok';
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ id: 3, email: 'u@x.com' });
      (apiClient.getUserMemberships as jest.Mock).mockRejectedValue(new Error('Network error'));

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual({ id: 3, email: 'u@x.com' });
      expect(state.userLoaded).toBe(true);
      expect(state.userLoading).toBe(false);
      // Memberships remain empty on failure
      expect(state.memberships).toEqual([]);
      expect(state.orgRoleMap.size).toBe(0);
    });

    it('does not fetch memberships when user has no id', async () => {
      mockCookies['csrf_token'] = 'tok';
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValue({ email: 'u@x.com' });

      await useAuthStore.getState().loadUser();

      expect(apiClient.getUserMemberships).not.toHaveBeenCalled();
    });
  });

  describe('unauthorized callback', () => {
    it('registers a callback via setOnUnauthorized at module load', () => {
      expect(getUnauthorizedCallback()).not.toBeNull();
    });

    it('clears auth state and localStorage when triggered', () => {
      // Set up authenticated state
      useAuthStore.setState({
        user: { id: 1, email: 'test@test.com' },
        isAuthenticated: true,
        userLoaded: true,
        memberships: [{ user_id: 1, organization_id: 5, role: 'admin' as const }],
        orgRoleMap: new Map([[5, 'admin' as const]]),
      });
      localStorage.setItem('selectedOrgId', '5');

      // Trigger the unauthorized callback captured at module load
      getUnauthorizedCallback()();

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.userLoaded).toBe(false);
      expect(state.memberships).toEqual([]);
      expect(state.orgRoleMap.size).toBe(0);
      expect(localStorage.getItem('selectedOrgId')).toBeNull();
    });

    it('calls setHasSession(false) to prevent refresh loops', () => {
      getUnauthorizedCallback()();

      expect(apiClient.setHasSession).toHaveBeenCalledWith(false);
    });
  });

  describe('logout clears memberships and orgRoleMap', () => {
    it('resets memberships and orgRoleMap on logout', async () => {
      (apiClient.logout as jest.Mock).mockResolvedValue(undefined);

      useAuthStore.setState({
        user: { id: 1, email: 'test@test.com' },
        isAuthenticated: true,
        userLoaded: true,
        memberships: [{ user_id: 1, organization_id: 5, role: 'admin' as const }],
        orgRoleMap: new Map([[5, 'admin' as const]]),
      });

      await useAuthStore.getState().logout();

      const state = useAuthStore.getState();
      expect(state.memberships).toEqual([]);
      expect(state.orgRoleMap.size).toBe(0);
    });
  });

  describe('no localStorage token usage', () => {
    it('should not store token in localStorage on login', async () => {
      (apiClient.login as jest.Mock).mockResolvedValue({ expires_in: 3600 });
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
