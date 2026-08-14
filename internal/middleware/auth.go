package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/problem"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// AuthMiddleware authenticates requests against server-side sessions.
//
// A request presents a session in one of two ways:
//  1. Browser: HttpOnly `session` cookie (preferred). CSRF applies.
//  2. CLI / server-to-server: `Authorization: Bearer <raw>` with the same
//     value the login response placed in the cookie. CSRF does not apply
//     because browsers never auto-attach this header.
//
// The raw value is hashed with sha256 before looking up the sessions row, so
// the stored id column is not a usable credential if the DB is leaked.
type AuthMiddleware struct {
	sessionStore  store.SessionStorer
	secureCookies bool
}

// NewAuthMiddleware constructs the middleware. The session store is required;
// there is no opt-out mode.
func NewAuthMiddleware(sessionStore store.SessionStorer) *AuthMiddleware {
	return &AuthMiddleware{sessionStore: sessionStore}
}

// SetSecureCookies controls whether cookies cleared by the middleware carry
// the Secure attribute. Must match the value used by the auth handler so the
// browser treats the cleared Set-Cookie as applying to the same cookie.
func (m *AuthMiddleware) SetSecureCookies(v bool) { m.secureCookies = v }

// Auth cookie names. Duplicated from internal/handlers/auth.go to avoid a
// middleware→handlers import; they are part of the HTTP contract and change
// together.
const (
	authCookieSession = "session"
	authCookieCSRF    = "csrf_token"
)

// clearAuthCookies expires the auth cookies so a client holding a session
// value the server has just rejected stops sending it on subsequent requests.
func (m *AuthMiddleware) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieSession, "", -1, "/", "", m.secureCookies, true)
	c.SetCookie(authCookieCSRF, "", -1, "/", "", m.secureCookies, false)
}

// extractRawToken returns the session value presented by the client, or an
// empty string when none is present.
func extractRawToken(c *gin.Context) string {
	if cookie, err := c.Cookie(authCookieSession); err == nil && cookie != "" {
		return cookie
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// RequireAuth returns a Gin middleware that validates the presented session
// and populates `ctxkeys.UserID` and `ctxkeys.UserEmail` for downstream
// handlers.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractRawToken(c)
		if raw == "" {
			slog.Warn("Auth failed: no session cookie or authorization header", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "authorization required")
			c.Abort()
			return
		}

		idHash := store.HashSessionToken(raw)
		lookup, err := m.sessionStore.Lookup(c.Request.Context(), idHash)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				slog.Warn("Auth failed: session not found or expired", "ip", c.ClientIP(), "path", c.Request.URL.Path)
				m.clearAuthCookies(c)
				problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "invalid or expired session")
				c.Abort()
				return
			}
			slog.Error("Failed to look up session", "error", err, "ip", c.ClientIP())
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "internal server error")
			c.Abort()
			return
		}

		if !lookup.UserActive {
			// Defense-in-depth: a deactivated user must not be able to use
			// their session even if a cleanup job didn't run. Also best-effort
			// prune the row so the next request is cheaper.
			_ = m.sessionStore.Delete(c.Request.Context(), idHash)
			slog.Warn("Auth failed: user is inactive", "user_id", lookup.UserID, "ip", c.ClientIP())
			m.clearAuthCookies(c)
			problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "account disabled")
			c.Abort()
			return
		}

		c.Set(ctxkeys.UserID, lookup.UserID)
		c.Set(ctxkeys.UserEmail, lookup.UserEmail)
		// Stash the id hash so handlers such as self-service password change
		// can scope "keep this session" correctly.
		c.Set(ctxkeys.SessionIDHash, idHash)
		c.Next()
	}
}
