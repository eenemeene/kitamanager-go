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
