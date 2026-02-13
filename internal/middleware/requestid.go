package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header name for request IDs.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the gin context key for the request ID.
	RequestIDKey = "requestID"
)

// RequestID returns a middleware that generates a unique request ID for each request.
// If the incoming request already has an X-Request-ID header, it is reused.
// The request ID is set in the response header and stored in the gin context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)

		c.Next()
	}
}
