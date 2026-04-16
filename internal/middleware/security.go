package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodySize is the maximum allowed request body size (10MB).
// This must be at least as large as the largest per-endpoint upload limit
// (e.g., handlers.MaxUploadSize = 5MB) to avoid the global middleware
// rejecting uploads before the handler's own validation runs.
const MaxRequestBodySize = 10 << 20 // 10MB

// SecurityHeaders adds common security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enforce HTTPS via HSTS (1 year, include subdomains, allow preload)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content Security Policy — restrict resource loading to same origin
		c.Header("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS filter in older browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict permissions/features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Prevent caching of API responses containing sensitive data
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
