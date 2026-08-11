// Tests for ApiClient's axios interceptors. The existing client.test.ts
// stubs interceptors to no-ops, which means CSRF, 401-handling, and 429
// enrichment have zero coverage. Here we capture the handlers passed to
// `interceptors.{request,response}.use()` and exercise them directly.

import type { AxiosError, InternalAxiosRequestConfig } from 'axios';

// `var` declarations are hoisted to the top of the module, which is
// required because `jest.mock(...)` factories run before any
// non-hoisted declarations (`const`/`let`) are initialised. The mock
// stuffs the captured handlers into this object; tests below read it
// after the module load completes.
var captured: {
  requestSuccess: ((c: InternalAxiosRequestConfig) => InternalAxiosRequestConfig) | null;
  requestError: ((e: unknown) => unknown) | null;
  responseSuccess: ((r: unknown) => unknown) | null;
  responseError: ((e: AxiosError) => unknown) | null;
};

jest.mock('axios', () => {
  captured = {
    requestSuccess: null,
    requestError: null,
    responseSuccess: null,
    responseError: null,
  };
  const instance = {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    interceptors: {
      request: {
        use: (success: never, error: never) => {
          captured.requestSuccess = success;
          captured.requestError = error;
        },
      },
      response: {
        use: (success: never, error: never) => {
          captured.responseSuccess = success;
          captured.responseError = error;
        },
      },
    },
  };
  return { __esModule: true, default: { create: jest.fn(() => instance) } };
});

// Mock the cookie helper so we can drive what `getCSRFToken()` returns.
jest.mock('@/lib/utils', () => ({
  ...jest.requireActual('@/lib/utils'),
  getCookie: jest.fn(),
}));

import { apiClient } from '../client';
import { getCookie } from '@/lib/utils';

const getCookieMock = getCookie as jest.Mock;

function makeConfig(
  overrides: Partial<InternalAxiosRequestConfig> = {}
): InternalAxiosRequestConfig {
  return {
    method: 'get',
    url: '/api/v1/test',
    headers: {} as InternalAxiosRequestConfig['headers'],
    ...overrides,
  } as InternalAxiosRequestConfig;
}

function makeError(
  status: number,
  url: string,
  method: 'get' | 'post' | 'put' | 'delete' = 'get',
  extras: Record<string, unknown> = {}
): AxiosError {
  return {
    config: { url, method, headers: {} } as InternalAxiosRequestConfig,
    response: {
      status,
      headers: {},
      data: extras,
      statusText: '',
      config: {} as InternalAxiosRequestConfig,
    },
    isAxiosError: true,
    name: 'AxiosError',
    message: '',
    toJSON: () => ({}),
  } as AxiosError;
}

describe('ApiClient interceptors', () => {
  // Force module-load eagerly so the interceptors register before tests
  // run. Reading `apiClient` is enough to trigger the constructor.
  beforeAll(() => {
    void apiClient;
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('request interceptor — CSRF token attachment', () => {
    it('adds the X-CSRF-Token header to POST when a csrf_token cookie is set', () => {
      // Critical: state-changing requests must carry the CSRF header so
      // the backend's middleware/csrf.go check passes. Without it, every
      // POST would 403 silently.
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: 'post' }));
      expect(cfg.headers!['X-CSRF-Token']).toBe('csrf-abc');
    });

    it('adds the header to PUT requests', () => {
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: 'put' }));
      expect(cfg.headers!['X-CSRF-Token']).toBe('csrf-abc');
    });

    it('adds the header to DELETE requests', () => {
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: 'delete' }));
      expect(cfg.headers!['X-CSRF-Token']).toBe('csrf-abc');
    });

    it('does NOT add the header to GET requests', () => {
      // GETs are idempotent and don't change state, so they don't need
      // CSRF. Adding the header would still work, but we want to keep
      // the surface minimal so the test catches a regression that
      // accidentally sends CSRF on every request.
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: 'get' }));
      expect(cfg.headers!['X-CSRF-Token']).toBeUndefined();
    });

    it('does NOT add the header to HEAD or OPTIONS requests', () => {
      getCookieMock.mockReturnValue('csrf-abc');
      const head = captured.requestSuccess!(makeConfig({ method: 'head' }));
      expect(head.headers!['X-CSRF-Token']).toBeUndefined();
      const options = captured.requestSuccess!(makeConfig({ method: 'options' }));
      expect(options.headers!['X-CSRF-Token']).toBeUndefined();
    });

    it('skips the header silently when no csrf_token cookie exists', () => {
      // First-page-load case: the user hasn't logged in yet, no cookie.
      // We must not throw; the request just goes out without the header
      // and the backend will reject if it was needed.
      getCookieMock.mockReturnValue(null);
      const cfg = captured.requestSuccess!(makeConfig({ method: 'post' }));
      expect(cfg.headers!['X-CSRF-Token']).toBeUndefined();
    });

    it('handles undefined method (defaults to lowercase no-op)', () => {
      // axios may pass a config without `method` set when the caller
      // doesn't specify; the interceptor must not crash on that.
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: undefined as unknown as 'get' }));
      // No method → not in the state-changing list → no header.
      expect(cfg.headers!['X-CSRF-Token']).toBeUndefined();
    });

    it('handles uppercase methods (case-insensitive comparison)', () => {
      // axios sometimes sets method as uppercase ('POST') after user
      // input. The interceptor must lowercase it to match the
      // exclusion list. A regression that compared raw values would
      // accept all uppercase methods (including GET).
      getCookieMock.mockReturnValue('csrf-abc');
      const cfg = captured.requestSuccess!(makeConfig({ method: 'POST' as unknown as 'post' }));
      expect(cfg.headers!['X-CSRF-Token']).toBe('csrf-abc');
    });
  });

  describe('response interceptor — 401 handling', () => {
    let unauthCb: jest.Mock;

    beforeEach(() => {
      unauthCb = jest.fn();
      apiClient.setOnUnauthorized(unauthCb);
    });

    it('fires the unauthorized callback for a generic 401 (e.g. expired session)', async () => {
      const promise = captured.responseError!(makeError(401, '/api/v1/users/me'));
      await expect(promise).rejects.toBeDefined();
      expect(unauthCb).toHaveBeenCalledTimes(1);
    });

    it('does NOT fire the callback for a 401 on /login (bad creds, not session loss)', async () => {
      // /login 401 means "wrong password" — the user isn't logged in
      // yet, there's nothing to clear. Firing the callback would
      // redirect them to /login again, hiding the error message.
      const promise = captured.responseError!(makeError(401, '/api/v1/login', 'post'));
      await expect(promise).rejects.toBeDefined();
      expect(unauthCb).not.toHaveBeenCalled();
    });

    it('does NOT fire the callback for a 401 on /logout', async () => {
      // /logout 401 just means the session was already gone — silently
      // not-our-problem.
      const promise = captured.responseError!(makeError(401, '/api/v1/logout', 'post'));
      await expect(promise).rejects.toBeDefined();
      expect(unauthCb).not.toHaveBeenCalled();
    });

    it('does NOT fire the callback for a 401 on /auth/mfa/verify', async () => {
      // 401 on MFA verify means "wrong code", not "session gone".
      // Wiping the auth store would yank the user back to /login mid-
      // dialog and they'd lose the partially-completed MFA flow.
      const promise = captured.responseError!(makeError(401, '/api/v1/auth/mfa/verify', 'post'));
      await expect(promise).rejects.toBeDefined();
      expect(unauthCb).not.toHaveBeenCalled();
    });

    it('does NOT fire the callback for a 401 on a factor-mutation endpoint (step-up password failure)', async () => {
      // POST /users/:id/factors and friends require step-up password —
      // a 401 means "wrong step-up password," not "session gone."
      // See webauthn.spec.ts for the regression scenario.
      const promise = captured.responseError!(makeError(401, '/api/v1/users/me/factors', 'post'));
      await expect(promise).rejects.toBeDefined();
      expect(unauthCb).not.toHaveBeenCalled();
    });

    it('does NOT fire the callback for a 401 on factor activation/regeneration', async () => {
      const cases = [
        { url: '/api/v1/users/me/factors/42/activate', method: 'post' as const },
        { url: '/api/v1/users/me/factors/42/regenerate', method: 'post' as const },
        { url: '/api/v1/users/me/factors/42', method: 'delete' as const },
      ];
      for (const c of cases) {
        unauthCb.mockClear();
        const p = captured.responseError!(makeError(401, c.url, c.method));
        await expect(p).rejects.toBeDefined();
        expect(unauthCb).not.toHaveBeenCalled();
      }
    });

    it('DOES fire the callback for a GET on a factor URL (genuine session loss)', async () => {
      // The factor-mutation rule excludes ONLY non-GET methods —
      // a GET 401 on the same path is a real session-expired event
      // (the user opened their factors page after the cookie expired)
      // and should redirect to login.
      const p = captured.responseError!(makeError(401, '/api/v1/users/me/factors', 'get'));
      await expect(p).rejects.toBeDefined();
      expect(unauthCb).toHaveBeenCalledTimes(1);
    });

    it('does not throw when no onUnauthorized callback is set', async () => {
      // Defensive: the auth store registers the callback during app
      // bootstrap, but a transient race during HMR or test-runtime
      // could fire a 401 before the callback exists.
      apiClient.setOnUnauthorized(undefined as unknown as () => void);
      const p = captured.responseError!(makeError(401, '/api/v1/users/me'));
      await expect(p).rejects.toBeDefined();
    });

    it('always rejects (never swallows) the error', async () => {
      // Even the auth-endpoint case must propagate the error — the
      // login form needs to see "Invalid credentials" on screen.
      const err = makeError(401, '/api/v1/login', 'post', { message: 'bad creds' });
      await expect(captured.responseError!(err)).rejects.toBe(err);
    });
  });

  describe('response interceptor — 429 enrichment', () => {
    it('enriches the error data with a "Rate limit exceeded" message including Retry-After seconds', async () => {
      // 429s without a message produce an opaque "Network error" toast.
      // The interceptor injects a usable message so the user knows
      // when to retry.
      const err = makeError(429, '/api/v1/login', 'post', {});
      err.response!.headers = { 'retry-after': '30' };
      await expect(captured.responseError!(err)).rejects.toBe(err);
      const data = err.response!.data as { message?: string };
      expect(data.message).toMatch(/30 seconds/);
    });

    it('falls back to a generic message when Retry-After is missing', async () => {
      const err = makeError(429, '/api/v1/login', 'post', {});
      // no retry-after header
      await expect(captured.responseError!(err)).rejects.toBe(err);
      const data = err.response!.data as { message?: string };
      expect(data.message).toBe('Rate limit exceeded. Please try again later.');
    });

    it('does NOT overwrite a backend-provided message', async () => {
      // If the backend went to the trouble of supplying a specific
      // 429 message, we must not clobber it. (E.g. account lockout
      // could surface "Locked for 5 minutes" from the backend.)
      const err = makeError(429, '/api/v1/login', 'post', { message: 'Account locked' });
      await expect(captured.responseError!(err)).rejects.toBe(err);
      const data = err.response!.data as { message?: string };
      expect(data.message).toBe('Account locked');
    });

    it('does not fire the unauthorized callback on 429', async () => {
      // 429 is short-circuited before the 401 logic runs. Crucial:
      // a rate-limited login attempt mustn't wipe the auth store.
      const cb = jest.fn();
      apiClient.setOnUnauthorized(cb);
      const err = makeError(429, '/api/v1/users/me');
      await expect(captured.responseError!(err)).rejects.toBeDefined();
      expect(cb).not.toHaveBeenCalled();
    });
  });

  describe('other status codes pass through', () => {
    it('rejects 500s without modifying the error', async () => {
      const err = makeError(500, '/api/v1/anything', 'get', { message: 'oops' });
      const cb = jest.fn();
      apiClient.setOnUnauthorized(cb);
      await expect(captured.responseError!(err)).rejects.toBe(err);
      expect(cb).not.toHaveBeenCalled();
    });

    it('rejects 403s without firing the unauthorized callback', async () => {
      // 403 is "you are logged in but not allowed" — clearing auth
      // state would be wrong (it's still a valid session).
      const err = makeError(403, '/api/v1/admin/users');
      const cb = jest.fn();
      apiClient.setOnUnauthorized(cb);
      await expect(captured.responseError!(err)).rejects.toBe(err);
      expect(cb).not.toHaveBeenCalled();
    });
  });
});
