package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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
// The CSRF token is derived from the access token via HMAC, binding it to the session.
// This prevents cookie-injection attacks where an attacker sets their own CSRF cookie.
type CSRFMiddleware struct {
	jwtSecret string
}

// NewCSRFMiddleware creates a new CSRF middleware instance.
func NewCSRFMiddleware(jwtSecret string) *CSRFMiddleware {
	return &CSRFMiddleware{jwtSecret: jwtSecret}
}

// ComputeCSRFToken derives a CSRF token from the access token using HMAC-SHA256.
// This binds the CSRF token to the specific session, preventing cookie injection.
func ComputeCSRFToken(accessToken, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))
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
		accessToken, cookieErr := c.Cookie("access_token")
		authHeader := c.GetHeader("Authorization")

		// If no cookie auth is used (pure API client with header auth), skip CSRF
		if cookieErr != nil && authHeader != "" {
			c.Next()
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

		// Compute expected CSRF token from the access token cookie
		expectedCSRF := ComputeCSRFToken(accessToken, m.jwtSecret)

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(csrfHeader), []byte(expectedCSRF)) != 1 {
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
