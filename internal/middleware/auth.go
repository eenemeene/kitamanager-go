package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: jwtSecret}
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
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "authorization required",
				})
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "invalid authorization header format",
				})
				c.Abort()
				return
			}
			tokenString = parts[1]
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid token claims",
			})
			c.Abort()
			return
		}

		// Verify this is an access token (not a refresh token)
		// For backwards compatibility, accept tokens without type claim
		if tokenType, exists := claims["type"]; exists {
			if tokenType != "access" {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Code:    apperror.CodeUnauthorized,
					Message: "invalid token type",
				})
				c.Abort()
				return
			}
		}

		// JWT numbers are parsed as float64, convert to uint
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "invalid user id in token",
			})
			c.Abort()
			return
		}

		c.Set("userID", uint(userIDFloat))
		c.Set("userEmail", claims["email"])
		c.Next()
	}
}
