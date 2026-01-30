import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Paths that don't require authentication
const publicPaths = ['/login'];

/**
 * Validate that a path is safe for redirect (not an open redirect vulnerability).
 * Only allows relative paths that start with / and don't contain protocol schemes.
 */
function isValidRedirectPath(path: string): boolean {
  // Must start with a single slash
  if (!path.startsWith('/')) return false;
  // Reject protocol-relative URLs (//example.com)
  if (path.startsWith('//')) return false;
  // Reject URLs with protocol schemes
  if (path.includes('://')) return false;
  // Reject paths that could be interpreted as absolute URLs
  if (path.includes('\\')) return false;
  return true;
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Check if the path is public
  const isPublicPath = publicPaths.some((path) => pathname.startsWith(path));

  // Check for CSRF token cookie (indicates authenticated session)
  // The access_token is HttpOnly, but csrf_token is readable
  const csrfToken = request.cookies.get('csrf_token')?.value;
  const isAuthenticated = !!csrfToken;

  // For public paths, if user is logged in, redirect to dashboard
  if (isPublicPath && isAuthenticated) {
    return NextResponse.redirect(new URL('/', request.url));
  }

  // For protected paths, if not authenticated, redirect to login
  if (!isPublicPath && !isAuthenticated) {
    const loginUrl = new URL('/login', request.url);
    // Only set 'from' parameter if it's a valid relative path
    if (isValidRedirectPath(pathname)) {
      loginUrl.searchParams.set('from', pathname);
    }
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - api routes (handled by API proxy)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico, etc.
     */
    '/((?!api|_next/static|_next/image|favicon.ico|.*\\..*|_next).*)',
  ],
};
