import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { proxy, config } from '../proxy';

jest.mock('next/server', () => {
  const redirect = jest.fn().mockReturnValue({ type: 'redirect' });
  const next = jest.fn().mockReturnValue({ type: 'next' });
  return {
    NextResponse: { redirect, next },
  };
});

function createMockRequest(pathname: string, csrfToken?: string): NextRequest {
  const url = new URL(`http://localhost:3000${pathname}`);
  return {
    nextUrl: url,
    url: url.toString(),
    cookies: {
      get: jest.fn((name: string) =>
        name === 'csrf_token' && csrfToken ? { value: csrfToken } : undefined
      ),
    },
  } as unknown as NextRequest;
}

describe('proxy function', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('allows authenticated user on protected path', () => {
    const result = proxy(createMockRequest('/dashboard', 'token'));
    expect(result).toEqual({ type: 'next' });
  });

  it('redirects unauthenticated user to login with from param', () => {
    proxy(createMockRequest('/dashboard'));
    const redirectUrl = (NextResponse.redirect as jest.Mock).mock.calls[0][0] as URL;
    expect(redirectUrl.pathname).toBe('/login');
    expect(redirectUrl.searchParams.get('from')).toBe('/dashboard');
  });

  it('redirects authenticated user away from login page', () => {
    proxy(createMockRequest('/login', 'token'));
    const redirectUrl = (NextResponse.redirect as jest.Mock).mock.calls[0][0] as URL;
    expect(redirectUrl.pathname).toBe('/');
  });

  it('allows unauthenticated user on login page', () => {
    const result = proxy(createMockRequest('/login'));
    expect(result).toEqual({ type: 'next' });
  });

  it('does not set from param for protocol-relative paths', () => {
    proxy(createMockRequest('//evil.com'));
    const redirectUrl = (NextResponse.redirect as jest.Mock).mock.calls[0][0] as URL;
    expect(redirectUrl.searchParams.has('from')).toBe(false);
  });

  it('does not set from param for backslash paths', () => {
    // URL constructor normalizes backslashes to forward slashes,
    // so we need to mock nextUrl directly to test isValidRedirectPath
    const req = createMockRequest('/path');
    // Override pathname to contain a backslash (bypassing URL normalization)
    Object.defineProperty(req.nextUrl, 'pathname', { value: '/path\\evil', writable: false });
    proxy(req);
    const redirectUrl = (NextResponse.redirect as jest.Mock).mock.calls[0][0] as URL;
    expect(redirectUrl.searchParams.has('from')).toBe(false);
  });

  it('exports a matcher config', () => {
    expect(config.matcher).toBeDefined();
    expect(config.matcher.length).toBeGreaterThan(0);
  });
});

// Test the path matching logic
describe('proxy path matching', () => {
  const publicPaths = ['/login'];

  function isPublicPath(pathname: string): boolean {
    return publicPaths.some((path) => pathname.startsWith(path));
  }

  describe('public paths', () => {
    it('identifies /login as public', () => {
      expect(isPublicPath('/login')).toBe(true);
    });

    it('identifies /login/callback as public', () => {
      expect(isPublicPath('/login/callback')).toBe(true);
    });

    it('identifies / as protected', () => {
      expect(isPublicPath('/')).toBe(false);
    });

    it('identifies /organizations as protected', () => {
      expect(isPublicPath('/organizations')).toBe(false);
    });

    it('identifies /government-funding-rates as protected', () => {
      expect(isPublicPath('/government-funding-rates')).toBe(false);
    });
  });

  describe('authentication flow', () => {
    function getAuthAction(
      pathname: string,
      hasToken: boolean
    ): 'redirect-to-dashboard' | 'redirect-to-login' | 'allow' {
      const isPublic = isPublicPath(pathname);

      if (isPublic && hasToken) {
        return 'redirect-to-dashboard';
      }

      if (!isPublic && !hasToken) {
        return 'redirect-to-login';
      }

      return 'allow';
    }

    it('redirects to dashboard when logged in and accessing login', () => {
      expect(getAuthAction('/login', true)).toBe('redirect-to-dashboard');
    });

    it('allows access to login when not logged in', () => {
      expect(getAuthAction('/login', false)).toBe('allow');
    });

    it('allows access to protected path when logged in', () => {
      expect(getAuthAction('/organizations', true)).toBe('allow');
    });

    it('redirects to login when accessing protected path without token', () => {
      expect(getAuthAction('/organizations', false)).toBe('redirect-to-login');
    });

    it('redirects to login when accessing dashboard without token', () => {
      expect(getAuthAction('/', false)).toBe('redirect-to-login');
    });

    it('allows access to dashboard when logged in', () => {
      expect(getAuthAction('/', true)).toBe('allow');
    });

    it('allows nested protected routes when logged in', () => {
      expect(getAuthAction('/organizations/1/employees', true)).toBe('allow');
    });

    it('redirects nested protected routes without token', () => {
      expect(getAuthAction('/organizations/1/employees', false)).toBe('redirect-to-login');
    });
  });
});
