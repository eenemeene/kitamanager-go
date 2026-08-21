import { create } from 'zustand';
import { apiClient } from '@/lib/api/client';
import type { User, LoginRequest, LoginResponse, Role, UserMembership } from '@/lib/api/types';
import { getCookie } from '@/lib/utils';
import { clearSessionCache } from '@/lib/api/session-cache';
import { useUiStore } from './ui-store';

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

  login: (credentials: LoginRequest) => Promise<LoginResponse>;
  hydrateAfterAuth: () => Promise<void>;
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

/**
 * Everything a departing user leaves behind in this tab.
 *
 * Logout and login are both soft navigations, so nothing here is torn down by a
 * remount -- the next user inherits it all. Three things had to go:
 *
 *   - The react-query cache, which still held the previous user's data and, for
 *     `['factors', 'me']` and `['me', 'sessions']`, held it under a key with no
 *     identity in it. The settings page would render the previous user's MFA
 *     factors and active sessions until the refetch landed.
 *   - The selected organization, which is persisted. This used to remove a key
 *     called `selectedOrgId` that nothing has ever written -- zustand persists
 *     the ui store under `ui-storage` -- so the removal was a no-op and the next
 *     user was redirected straight into the previous user's organization by the
 *     root page.
 *   - The organization list itself, so the selector cannot show a name from an
 *     account that is no longer signed in.
 */
function endSession() {
  clearSessionCache();
  useUiStore.getState().clearOrganizationSelection();
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

  // login POSTs /login and returns the discriminated response. On
  // `authenticated` the store hydrates user + memberships immediately;
  // on `mfa_required` the store state is LEFT ALONE — the page owns
  // the pending-token + MFA step, and only after a successful
  // /auth/mfa/verify does the caller invoke `hydrateAfterAuth` to
  // populate the store. This keeps "authentication state" and
  // "mid-authentication state" cleanly separate.
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response = await apiClient.login(credentials);
    if (response.status === 'authenticated') {
      await get().hydrateAfterAuth();
    }
    return response;
  },

  // hydrateAfterAuth is called by the login page once the session
  // cookie is in place — either directly after a non-MFA login or
  // after a successful /auth/mfa/verify. Splits the post-auth
  // hydration out of `login` so the MFA verify call site can reuse it.
  hydrateAfterAuth: async () => {
    set({ isAuthenticated: true });
    try {
      const userData = await apiClient.getCurrentUser();
      set({ user: userData, userLoaded: true });
      // Self-memberships: same reasoning as in loadUser. The
      // session cookie is already in place, so /me/memberships
      // resolves the user from the session — no userData.id guard
      // needed, which is also why we no longer test for it.
      try {
        const { memberships } = await apiClient.getMyMemberships();
        set({ memberships, orgRoleMap: buildOrgRoleMap(memberships) });
      } catch {
        // Non-critical: navigation falls back to global-only items.
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
    // Auth state first, then everything downstream of it. The org selector
    // refetches whenever it sees an authenticated user with an empty
    // organization list, so emptying the list while `isAuthenticated` was still
    // true would fire one guaranteed-401 request on the way out.
    set({
      user: null,
      isAuthenticated: false,
      userLoaded: false,
      memberships: [],
      orgRoleMap: new Map(),
    });
    endSession();
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

      // Fetch memberships for role-based navigation. Use the self-route
      // (/me/memberships) — it does not require users:read, so it works
      // for staff and member users too. The admin-facing
      // /users/{userId}/memberships still exists for user-management
      // screens but isn't usable here.
      try {
        const { memberships } = await apiClient.getMyMemberships();
        set({ memberships, orgRoleMap: buildOrgRoleMap(memberships) });
      } catch {
        // Non-critical: navigation will silently fail open if memberships
        // can't be loaded. The sidebar then renders only global items.
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
  // Clear local state without calling logout endpoint (already unauthorized).
  if (typeof window !== 'undefined') {
    // Clear the JS-readable csrf_token cookie. Without this, hasAuthCookie()
    // keeps returning true after the server rejected our session, which
    // causes components to re-fire authenticated requests in an infinite
    // loop. The HttpOnly session cookie is cleared by the server on 401.
    document.cookie = 'csrf_token=; path=/; max-age=0; SameSite=Strict';
  }
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    userLoaded: false,
    memberships: [],
    orgRoleMap: new Map(),
  });
  endSession();
});
