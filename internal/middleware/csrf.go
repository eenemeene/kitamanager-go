package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

const (
	// CSRFHeaderName is the header name for the CSRF token
	CSRFHeaderName = "X-CSRF-Token"
	// CSRFCookieName is the cookie name for the CSRF token
	CSRFCookieName = "csrf_token"
)

// CSRFMiddleware validates CSRF tokens for state-changing requests.
// It checks that the X-CSRF-Token header matches the csrf_token cookie.
// Safe methods (GET, HEAD, OPTIONS) are allowed without CSRF validation.
type CSRFMiddleware struct{}

// NewCSRFMiddleware creates a new CSRF middleware instance.
func NewCSRFMiddleware() *CSRFMiddleware {
	return &CSRFMiddleware{}
}

// ValidateCSRF returns a Gin middleware handler that validates CSRF tokens.
func (m *CSRFMiddleware) ValidateCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Safe methods don't require CSRF validation
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Check if request has cookie authentication
		// If using Authorization header only, skip CSRF check (for API clients)
		_, cookieErr := c.Cookie("access_token")
		authHeader := c.GetHeader("Authorization")

		// If no cookie auth is used (pure API client with header auth), skip CSRF
		if cookieErr != nil && authHeader != "" {
			c.Next()
			return
		}

		// For cookie-based auth, require CSRF token
		csrfCookie, err := c.Cookie(CSRFCookieName)
		if err != nil || csrfCookie == "" {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "csrf_error",
				Message: "CSRF token cookie missing",
			})
			c.Abort()
			return
		}

		csrfHeader := c.GetHeader(CSRFHeaderName)
		if csrfHeader == "" {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "csrf_error",
				Message: "CSRF token header missing",
			})
			c.Abort()
			return
		}

		// Constant-time comparison to prevent timing attacks
		if !secureCompare(csrfHeader, csrfCookie) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Code:    "csrf_error",
				Message: "CSRF token validation failed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isSafeMethod returns true for HTTP methods that are considered "safe"
// (i.e., they should not cause side effects).
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// secureCompare performs a constant-time comparison of two strings
// to prevent timing attacks.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
