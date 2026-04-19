package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodySize is the maximum allowed request body size (10MB).
// This must be at least as large as the largest per-endpoint upload limit
// (e.g., handlers.MaxUploadSize = 5MB) to avoid the global middleware
// rejecting uploads before the handler's own validation runs.
const MaxRequestBodySize = 10 << 20 // 10MB

// apiCSP is the Content Security Policy for JSON API responses. The API never
// serves HTML that loads scripts or styles, so every fetch directive is locked
// to 'none'. `connect-src 'self'` is allowed because some browser flows
// (e.g. Playwright tests that navigate directly to an API URL to seed cookies)
// end up with the API response as the page origin and then fetch() same-origin
// endpoints; the permission is harmless because the API never serves HTML that
// a user-controllable script could execute from.
const apiCSP = "default-src 'none'; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'none'; " +
	"form-action 'none'"

// swaggerCSP loosens the policy just for the Swagger UI asset bundle.
const swaggerCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// SecurityHeaders adds common security headers to responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS — 1 year, include subdomains, preload.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Swagger serves HTML + JS; everything else is JSON.
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Header("Content-Security-Policy", swaggerCSP)
		} else {
			c.Header("Content-Security-Policy", apiCSP)
		}

		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")

		// Modern guidance: disable the legacy XSS filter in Chrome/Edge.
		// A value of 1 can introduce its own XSS reflection bugs.
		c.Header("X-XSS-Protection", "0")

		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cache-Control", "no-store")

		c.Next()
	}
}

// BodySizeLimit limits the request body size to prevent DoS attacks
func BodySizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = &limitedReader{
			reader:    c.Request.Body,
			remaining: maxSize,
		}
		c.Next()
	}
}

// limitedReader wraps an io.ReadCloser and limits the number of bytes that can be read
type limitedReader struct {
	reader    interface{ Read([]byte) (int, error) }
	remaining int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, &requestTooLargeError{}
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.reader.Read(p)
	l.remaining -= int64(n)
	return n, err
}

func (l *limitedReader) Close() error {
	if closer, ok := l.reader.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

type requestTooLargeError struct{}

func (e *requestTooLargeError) Error() string {
	return "request body too large"
}

// DefaultRequestTimeout is the default timeout for request processing
const DefaultRequestTimeout = 30 * time.Second

// RequestTimeout adds a timeout context to each request.
// This ensures long-running database queries or external calls don't hang forever.
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context with timeout context
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
