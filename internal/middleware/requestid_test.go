package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	id := w.Header().Get(RequestIDHeader)
	if id == "" {
		t.Error("expected X-Request-ID header to be set")
	}
	if len(id) != 36 { // UUID v4 format
		t.Errorf("expected UUID format (36 chars), got %d chars: %q", len(id), id)
	}
}

func TestRequestID_ReusesExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "custom-request-id-123")
	r.ServeHTTP(w, req)

	id := w.Header().Get(RequestIDHeader)
	if id != "custom-request-id-123" {
		t.Errorf("expected reused request ID %q, got %q", "custom-request-id-123", id)
	}
}

func TestRequestID_SetsContextValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var contextID string
	r.GET("/test", func(c *gin.Context) {
		val, exists := c.Get(RequestIDKey)
		if !exists {
			t.Error("expected requestID in context")
			return
		}
		contextID = val.(string)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	headerID := w.Header().Get(RequestIDHeader)
	if contextID != headerID {
		t.Errorf("context ID %q != header ID %q", contextID, headerID)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest("GET", "/test", nil))

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/test", nil))

	id1 := w1.Header().Get(RequestIDHeader)
	id2 := w2.Header().Get(RequestIDHeader)
	if id1 == id2 {
		t.Errorf("expected unique request IDs, both got %q", id1)
	}
}
