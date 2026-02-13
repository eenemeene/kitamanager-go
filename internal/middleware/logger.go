package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogger returns a gin middleware for structured logging using slog.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Build log attributes
		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency", latency.String(),
			"latency_ms", latency.Milliseconds(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		// Get request ID if available
		if requestID, exists := c.Get(RequestIDKey); exists {
			attrs = append(attrs, "request_id", requestID)
		}

		// Get userID if available
		if userID, exists := c.Get("userID"); exists {
			attrs = append(attrs, "user_id", userID)
		}

		// Log based on status code
		switch {
		case status >= 500:
			slog.Error("Server error", attrs...)
		case status >= 400:
			slog.Warn("Client error", attrs...)
		default:
			slog.Info("Request", attrs...)
		}
	}
}
