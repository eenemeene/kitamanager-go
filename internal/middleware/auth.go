package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type AuthMiddleware struct {
	jwtSecret     string
	tokenStore    store.TokenStorer
	secureCookies bool
}

func NewAuthMiddleware(jwtSecret string, tokenStore ...store.TokenStorer) *AuthMiddleware {
	m := &AuthMiddleware{jwtSecret: jwtSecret}
	if len(tokenStore) > 0 {
		m.tokenStore = tokenStore[0]
	}
	return m
}

// SetSecureCookies configures whether cleared auth cookies carry the Secure
// attribute. Must match the value used by the auth handler so the browser
// treats the clearing Set-Cookie as applying to the same cookie.
func (m *AuthMiddleware) SetSecureCookies(v bool) {
	m.secureCookies = v
}

// Auth cookie names / paths. Duplicated from internal/handlers/auth.go to avoid
// a middleware→handlers import; they are part of the HTTP contract and change
// together.
const (
	authCookieAccess  = "access_token"
	authCookieRefresh = "refresh_token"
	authCookieCSRF    = "csrf_token"
	refreshCookiePath = "/api/v1/refresh"
)

// clearAuthCookies expires the auth cookies so a client holding tokens that
// the server has just rejected (e.g. signed with a rotated JWT secret) stops
// sending them on subsequent requests.
func (m *AuthMiddleware) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieAccess, "", -1, "/", "", m.secureCookies, true)
	c.SetCookie(authCookieRefresh, "", -1, refreshCookiePath, "", m.secureCookies, true)
	c.SetCookie(authCookieCSRF, "", -1, "/", "", m.secureCookies, false)
}

// HashToken computes the SHA-256 hash of a JWT token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// First, try to get token from HttpOnly cookie (preferred for security)
		if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			// Fall back to Authorization header for backwards compatibility and API clients
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				slog.Warn("Auth failed: no token or authorization header", "ip", c.ClientIP(), "path", c.Request.URL.Path)
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "authorization required",
				})
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				slog.Warn("Auth failed: invalid authorization header format", "ip", c.ClientIP(), "path", c.Request.URL.Path)
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "invalid authorization header format",
				})
				c.Abort()
				return
			}
			tokenString = parts[1]
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			slog.Warn("Auth failed: invalid token", "ip", c.ClientIP(), "path", c.Request.URL.Path, "error", err)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Warn("Auth failed: invalid token claims", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token claims",
			})
			c.Abort()
			return
		}

		// Verify this is an access token (not a refresh token)
		tokenType, _ := claims["type"].(string)
		if tokenType != "access" {
			slog.Warn("Auth failed: invalid token type", "ip", c.ClientIP(), "path", c.Request.URL.Path, "type", tokenType)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token type",
			})
			c.Abort()
			return
		}

		// Defense-in-depth: require exp claim even though jwt.Parse validates it when present.
		// Without this check, a token crafted without an exp claim would be accepted indefinitely.
		if _, hasExp := claims["exp"]; !hasExp {
			slog.Warn("Auth failed: missing exp claim", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token",
			})
			c.Abort()
			return
		}

		// JWT numbers are parsed as float64, convert to uint with bounds checking
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok || userIDFloat <= 0 || userIDFloat > math.MaxUint32 || userIDFloat != math.Trunc(userIDFloat) {
			slog.Warn("Auth failed: invalid user id in token", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid user id in token",
			})
			c.Abort()
			return
		}

		userID := uint(userIDFloat)

		// Extract iat for user-wide revocation check. Tokens issued by this
		// codebase always carry iat (see service.generateAccessToken); if it
		// is missing, the token is malformed and we fail closed.
		iatFloat, ok := claims["iat"].(float64)
		if !ok || iatFloat <= 0 {
			slog.Warn("Auth failed: missing iat claim", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			m.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token",
			})
			c.Abort()
			return
		}
		tokenIssuedAt := time.Unix(int64(iatFloat), 0).UTC()

		// Check if token has been revoked
		if m.tokenStore != nil {
			tokenHash := HashToken(tokenString)
			revoked, err := m.tokenStore.IsRevoked(c.Request.Context(), tokenHash)
			if err != nil {
				slog.Error("Failed to check token revocation", "error", err, "ip", c.ClientIP())
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Code:    apperror.CodeInternal,
					Message: "internal server error",
				})
				c.Abort()
				return
			}
			if revoked {
				m.clearAuthCookies(c)
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "token has been revoked",
				})
				c.Abort()
				return
			}

			// Revoke only tokens issued before the last RevokeAllForUser cutoff;
			// fresher tokens issued after a password change must keep working.
			userRevoked, err := m.tokenStore.IsUserRevokedSince(c.Request.Context(), userID, tokenIssuedAt)
			if err != nil {
				slog.Error("Failed to check user token revocation", "error", err, "ip", c.ClientIP())
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Code:    apperror.CodeInternal,
					Message: "internal server error",
				})
				c.Abort()
				return
			}
			if userRevoked {
				m.clearAuthCookies(c)
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "token has been revoked",
				})
				c.Abort()
				return
			}
		}

		c.Set(ctxkeys.UserID, userID)
		c.Set(ctxkeys.UserEmail, claims["email"])
		c.Next()
	}
}
