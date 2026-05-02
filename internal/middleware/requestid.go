package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header name for request IDs.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the gin context key for the request ID.
	RequestIDKey = "requestID"
)

// requestIDContextKey is the type used as the context.Context key so
// downstream services can read the request id via RequestIDFromContext
// without importing gin. The unexported struct type guarantees no
// collision with context keys from other packages.
type requestIDContextKey struct{}

// clientIPContextKey is the context.Context key for the request's
// client IP. Stamped by the same RequestID middleware that handles
// request ids — wiring one middleware (not two) keeps the
// composition simple. Used by AuditService.log() as a fallback IP
// source for callers that don't take an ipAddress argument
// (e.g. FactorService.LogFactor*). Closes review finding L3.
type clientIPContextKey struct{}

// RequestID returns a middleware that generates a unique request ID for
// each request. If the incoming request already has an X-Request-ID
// header, it is reused.
//
// The id is:
//   - written to the response header X-Request-ID so callers can
//     correlate their client-side logs;
//   - stored in the gin context under RequestIDKey for handler-side
//     access;
//   - propagated onto c.Request.Context() so service-layer code that
//     only sees context.Context (not gin.Context) can read it via
//     RequestIDFromContext — this is the path the AuditService uses
//     to stamp every audit row with the request id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)

		ctx := context.WithValue(c.Request.Context(), requestIDContextKey{}, id)
		// Also stash the client IP so service-layer audit emissions
		// that don't take an ipAddress argument can still record it
		// (L3). c.ClientIP() honours the framework's trusted-proxy
		// chain and falls back to RemoteAddr.
		ctx = context.WithValue(ctx, clientIPContextKey{}, c.ClientIP())
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequestIDFromContext returns the request id previously stamped by
// the RequestID middleware, or "" if no id is present (non-HTTP
// callers: seed imports, background jobs, tests that did not route
// through middleware).
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

// ContextWithRequestIDForTest stamps an arbitrary request id onto a
// context for use in unit tests that exercise code downstream of the
// middleware without actually booting a Gin router. Exported so the
// service package and others can write focused tests against the
// request-id plumbing; the name carries "ForTest" because production
// code has no business constructing ids by hand — only the middleware
// should.
func ContextWithRequestIDForTest(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

// ClientIPFromContext returns the client IP previously stamped by the
// RequestID middleware, or "" if no IP is present (non-HTTP callers:
// seed imports, background jobs, tests that did not route through
// middleware).
func ClientIPFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	ip, _ := ctx.Value(clientIPContextKey{}).(string)
	return ip
}

// ContextWithClientIPForTest is the test-only counterpart of
// ContextWithRequestIDForTest for the client IP slot.
func ContextWithClientIPForTest(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPContextKey{}, ip)
}
