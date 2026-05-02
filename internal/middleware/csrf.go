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
	// CSRFHeaderName is the header name for the CSRF token.
	CSRFHeaderName = "X-CSRF-Token"
	// CSRFCookieName is the cookie name for the CSRF token.
	CSRFCookieName = "csrf_token"
	// sessionCookieName duplicates handlers/auth.go's constant. Kept here to
	// avoid a middleware→handlers import; they are part of the HTTP contract
	// and change together.
	sessionCookieName = "session"
)

// CSRFMiddleware validates CSRF tokens for state-changing requests.
// The CSRF token is derived from the session cookie value via HMAC, binding
// it to the specific session. An attacker who injects a csrf_token cookie
// they control cannot produce a matching session-derived value.
type CSRFMiddleware struct {
	serverSecret string
}

// NewCSRFMiddleware creates a new CSRF middleware instance.
// `serverSecret` is a process-wide secret used for the HMAC
// derivation. Sourced from cfg.CSRFHMACKey (CSRF_HMAC_KEY env var),
// which falls back to JWT_SECRET when unset to keep existing
// deployments working — see config.Config.CSRFHMACKey for the
// rationale (audit finding C-M-3, security review 2026-05-01).
func NewCSRFMiddleware(serverSecret string) *CSRFMiddleware {
	return &CSRFMiddleware{serverSecret: serverSecret}
}

// ComputeCSRFToken derives a CSRF token from the session cookie value using
// HMAC-SHA256. This binds the CSRF token to the specific session, preventing
// cookie-injection attacks.
func ComputeCSRFToken(sessionValue, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionValue))
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateCSRF returns a Gin middleware handler that validates CSRF tokens.
func (m *CSRFMiddleware) ValidateCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Safe methods don't require CSRF validation.
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// CSRF is only a concern when the browser attaches credentials
		// automatically — i.e. when the request carries our session cookie.
		// Non-browser clients (CLI tools, curl, server-to-server) use
		// Authorization: Bearer and do not have browser-driven credential
		// attachment, so CSRF does not apply to them.
		sessionValue, cookieErr := c.Cookie(sessionCookieName)
		if cookieErr != nil {
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

		// Compute expected CSRF token from the session cookie.
		expectedCSRF := ComputeCSRFToken(sessionValue, m.serverSecret)

		// Constant-time comparison to prevent timing attacks.
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
