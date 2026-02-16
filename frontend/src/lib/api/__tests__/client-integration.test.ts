/**
 * Integration tests for ApiClient using MSW (Mock Service Worker).
 * Tests real Axios HTTP behavior including interceptors, token refresh, and error handling.
 *
 * @jest-environment node
 */
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

// Must import the real axios (no jest.mock)
// We create a fresh ApiClient per test by dynamically importing the module

const API_BASE = 'http://localhost/api/v1';

// Set env var before importing client so Axios uses absolute URLs
process.env.NEXT_PUBLIC_API_URL = 'http://localhost';

// Track handler invocations
let refreshCallCount: number;

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  server.resetHandlers();
  refreshCallCount = 0;
});
afterAll(() => server.close());

/**
 * Create a fresh ApiClient instance for each test.
 * This avoids stale state between tests.
 */
async function createFreshClient() {
  // Jest module isolation: re-import to get a fresh singleton
  jest.resetModules();
  // Ensure env var is set for each fresh import
  process.env.NEXT_PUBLIC_API_URL = 'http://localhost';
  const mod = await import('../client');
  return mod.apiClient;
}

describe('ApiClient integration (MSW)', () => {
  describe('login', () => {
    it('returns expires_in from login response', async () => {
      server.use(
        http.post(`${API_BASE}/login`, () => {
          return HttpResponse.json({
            expires_in: 3600,
          });
        })
      );

      const client = await createFreshClient();
      const result = await client.login({ email: 'test@example.com', password: 'pass' });

      expect(result.expires_in).toBe(3600);
    });
  });

  describe('token refresh on 401', () => {
    it('refreshes token via cookie and retries the original request on 401', async () => {
      let meCallCount = 0;

      server.use(
        // Login
        http.post(`${API_BASE}/login`, () => {
          return HttpResponse.json({
            expires_in: 3600,
          });
        }),
        // /me - first call returns 401, second succeeds
        http.get(`${API_BASE}/me`, () => {
          meCallCount++;
          if (meCallCount === 1) {
            return HttpResponse.json(
              { code: 'unauthorized', message: 'token expired' },
              { status: 401 }
            );
          }
          return HttpResponse.json({
            id: 1,
            email: 'test@example.com',
            name: 'Test User',
          });
        }),
        // Refresh endpoint - no body expected, refresh token comes via cookie
        http.post(`${API_BASE}/refresh`, () => {
          refreshCallCount++;
          return HttpResponse.json({
            expires_in: 3600,
          });
        })
      );

      const client = await createFreshClient();

      // Login to establish session
      await client.login({ email: 'test@example.com', password: 'pass' });

      // Call /me which will 401 -> refresh -> retry
      const user = await client.getCurrentUser();

      expect(user.name).toBe('Test User');
      expect(refreshCallCount).toBe(1);
      expect(meCallCount).toBe(2); // First 401, then retry succeeds
    });

    it('calls onUnauthorized when refresh fails', async () => {
      const onUnauthorized = jest.fn();

      server.use(
        http.post(`${API_BASE}/login`, () => {
          return HttpResponse.json({
            expires_in: 3600,
          });
        }),
        http.get(`${API_BASE}/me`, () => {
          return HttpResponse.json(
            { code: 'unauthorized', message: 'token expired' },
            { status: 401 }
          );
        }),
        http.post(`${API_BASE}/refresh`, () => {
          return HttpResponse.json(
            { code: 'unauthorized', message: 'refresh token expired' },
            { status: 401 }
          );
        })
      );

      const client = await createFreshClient();
      client.setOnUnauthorized(onUnauthorized);

      await client.login({ email: 'test@example.com', password: 'pass' });

      await expect(client.getCurrentUser()).rejects.toThrow();
      expect(onUnauthorized).toHaveBeenCalled();
    });

    it('calls onUnauthorized immediately when no session is available', async () => {
      const onUnauthorized = jest.fn();

      server.use(
        http.get(`${API_BASE}/me`, () => {
          return HttpResponse.json(
            { code: 'unauthorized', message: 'not authenticated' },
            { status: 401 }
          );
        })
      );

      const client = await createFreshClient();
      client.setOnUnauthorized(onUnauthorized);

      await expect(client.getCurrentUser()).rejects.toThrow();
      expect(onUnauthorized).toHaveBeenCalled();
    });
  });

  describe('429 rate limit handling', () => {
    it('enriches 429 error with user-friendly message', async () => {
      server.use(
        http.post(`${API_BASE}/organizations`, () => {
          return HttpResponse.json({ code: 'rate_limit_exceeded' }, { status: 429 });
        })
      );

      const client = await createFreshClient();

      try {
        await client.createOrganization({ name: 'Test Org' } as never);
        fail('Should have thrown');
      } catch (error: unknown) {
        const axiosError = error as { response?: { data?: { message?: string } } };
        expect(axiosError.response?.data?.message).toContain('Rate limit exceeded');
      }
    });

    it('includes Retry-After in 429 message when header is present', async () => {
      server.use(
        http.post(`${API_BASE}/organizations`, () => {
          return HttpResponse.json(
            { code: 'rate_limit_exceeded' },
            { status: 429, headers: { 'Retry-After': '30' } }
          );
        })
      );

      const client = await createFreshClient();

      try {
        await client.createOrganization({ name: 'Test Org' } as never);
        fail('Should have thrown');
      } catch (error: unknown) {
        const axiosError = error as { response?: { data?: { message?: string } } };
        expect(axiosError.response?.data?.message).toContain('30 seconds');
      }
    });
  });

  describe('logout clears session', () => {
    it('clears session on logout so 401 triggers onUnauthorized', async () => {
      const onUnauthorized = jest.fn();

      server.use(
        http.post(`${API_BASE}/login`, () => {
          return HttpResponse.json({
            expires_in: 3600,
          });
        }),
        http.post(`${API_BASE}/logout`, () => {
          return HttpResponse.json({ message: 'logged out' });
        }),
        http.get(`${API_BASE}/me`, () => {
          return HttpResponse.json(
            { code: 'unauthorized', message: 'not authenticated' },
            { status: 401 }
          );
        })
      );

      const client = await createFreshClient();
      client.setOnUnauthorized(onUnauthorized);

      await client.login({ email: 'test@example.com', password: 'pass' });
      await client.logout();

      // After logout, session should be cleared, so 401 triggers onUnauthorized directly
      await expect(client.getCurrentUser()).rejects.toThrow();
      expect(onUnauthorized).toHaveBeenCalled();
    });
  });
});
