package middleware

import (
	"context"
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

// Tests the ctx-plumbing half of the RequestID middleware: downstream
// code reading c.Request.Context() must see the same id that the
// header carries. This is the property the audit service relies on —
// middleware stamps ctx, auditService pulls from ctx, row carries the
// id. Without it, audit rows would all have empty request_ids even
// though the X-Request-ID header was present.
func TestRequestID_PropagatedOnRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var ctxID string
	r.GET("/test", func(c *gin.Context) {
		ctxID = RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "round-trip-42")
	r.ServeHTTP(w, req)

	if ctxID != "round-trip-42" {
		t.Errorf("RequestIDFromContext = %q, want %q", ctxID, "round-trip-42")
	}
	if got := w.Header().Get(RequestIDHeader); got != "round-trip-42" {
		t.Errorf("response header = %q, want %q", got, "round-trip-42")
	}
}

// RequestIDFromContext must be nil-safe and must return "" when
// called on a context that never routed through the middleware.
// Non-HTTP writers (seeds, background jobs, unit tests) rely on
// this semantic — they pass context.Background() and get back empty,
// which AuditService then persists as NULL.
func TestRequestIDFromContext_EmptyForBareContext(t *testing.T) {
	// Deliberately feed a nil Context to verify the defensive guard in
	// RequestIDFromContext. Staticcheck's SA1012 flags the pattern in
	// production code — here we are testing that the guard exists.
	var nilCtx context.Context //nolint:staticcheck // SA1012: exercising nil-safe path
	if got := RequestIDFromContext(nilCtx); got != "" {
		t.Errorf("nil ctx: got %q, want empty", got)
	}
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("bare ctx: got %q, want empty", got)
	}
}

// ContextWithRequestIDForTest is the escape hatch for unit tests
// that need a ctx carrying a specific id without booting a router.
func TestContextWithRequestIDForTest_RoundTrips(t *testing.T) {
	ctx := ContextWithRequestIDForTest(context.Background(), "synthetic-id")
	if got := RequestIDFromContext(ctx); got != "synthetic-id" {
		t.Errorf("round-trip: got %q, want %q", got, "synthetic-id")
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
