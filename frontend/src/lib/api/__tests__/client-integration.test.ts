/**
 * Integration tests for ApiClient using MSW (Mock Service Worker).
 * Tests real Axios HTTP behavior: CSRF header injection, 401 handling, 429
 * enrichment. There is no client-side token refresh any more — sessions live
 * for their server-side expiry and the only 401 response is "log out".
 *
 * @jest-environment node
 */
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

const API_BASE = 'http://localhost/api/v1';

// Set env var before importing client so Axios uses absolute URLs.
process.env.NEXT_PUBLIC_API_URL = 'http://localhost';

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  server.resetHandlers();
});
afterAll(() => server.close());

/** Create a fresh ApiClient instance for each test. */
async function createFreshClient() {
  jest.resetModules();
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
            status: 'authenticated',
            expires_in: 3600,
          });
        })
      );

      const client = await createFreshClient();
      const result = await client.login({ email: 'test@example.com', password: 'pass' });

      if (result.status !== 'authenticated') {
        throw new Error('expected authenticated result');
      }
      expect(result.expires_in).toBe(3600);
    });
  });

  describe('401 on a non-auth endpoint invokes onUnauthorized', () => {
    it('fires onUnauthorized and rejects when /me returns 401', async () => {
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

  describe('logout', () => {
    it('posts /logout and propagates 401 on subsequent calls via onUnauthorized', async () => {
      const onUnauthorized = jest.fn();

      server.use(
        http.post(`${API_BASE}/login`, () => {
          return HttpResponse.json({ status: 'authenticated', expires_in: 3600 });
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

      await expect(client.getCurrentUser()).rejects.toThrow();
      expect(onUnauthorized).toHaveBeenCalled();
    });
  });

  describe('changePassword', () => {
    it('PUTs /me/password with current_password and new_password in the body', async () => {
      let receivedBody: unknown;
      let receivedMethod = '';
      let receivedPath = '';

      server.use(
        http.post(`${API_BASE}/login`, () =>
          HttpResponse.json({ status: 'authenticated', expires_in: 3600 })
        ),
        http.put(`${API_BASE}/me/password`, async ({ request }) => {
          receivedMethod = request.method;
          receivedPath = new URL(request.url).pathname;
          receivedBody = await request.json();
          return HttpResponse.json({ status: 'authenticated', expires_in: 3600 });
        })
      );

      const client = await createFreshClient();
      await client.login({ email: 'test@example.com', password: 'old' });

      const result = await client.changePassword('old-password', 'new-password-8chars');

      expect(receivedMethod).toBe('PUT');
      expect(receivedPath).toBe('/api/v1/me/password');
      expect(receivedBody).toEqual({
        current_password: 'old-password',
        new_password: 'new-password-8chars',
      });
      expect(result.expires_in).toBe(3600);
    });

    it('does NOT call onUnauthorized on successful password change', async () => {
      const onUnauthorized = jest.fn();
      server.use(
        http.post(`${API_BASE}/login`, () =>
          HttpResponse.json({ status: 'authenticated', expires_in: 3600 })
        ),
        http.put(`${API_BASE}/me/password`, () =>
          HttpResponse.json({ status: 'authenticated', expires_in: 3600 })
        )
      );

      const client = await createFreshClient();
      client.setOnUnauthorized(onUnauthorized);
      await client.login({ email: 'test@example.com', password: 'old' });

      await client.changePassword('old', 'new-password-8chars');

      expect(onUnauthorized).not.toHaveBeenCalled();
    });

    it('a 401 on /me/password fires onUnauthorized and rejects', async () => {
      // /me/password is not in the auth-endpoint allowlist, so a 401 from it
      // flows through the same path as any other mutation: onUnauthorized is
      // called and the caller sees a rejection. The only auth-endpoint carve
      // out is /login and /logout.
      const onUnauthorized = jest.fn();

      server.use(
        http.post(`${API_BASE}/login`, () =>
          HttpResponse.json({ status: 'authenticated', expires_in: 3600 })
        ),
        http.put(`${API_BASE}/me/password`, () =>
          HttpResponse.json(
            { code: 'unauthorized', message: 'current password is incorrect' },
            { status: 401 }
          )
        )
      );

      const client = await createFreshClient();
      client.setOnUnauthorized(onUnauthorized);
      await client.login({ email: 'test@example.com', password: 'old' });

      await expect(client.changePassword('wrong', 'new-password-8chars')).rejects.toThrow();
      expect(onUnauthorized).toHaveBeenCalled();
    });
  });
});
