package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityHeaders_API(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	tests := []struct {
		header string
		want   string
	}{
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload"},
		{"Content-Security-Policy", apiCSP},
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "0"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "geolocation=(), microphone=(), camera=()"},
		{"Cache-Control", "no-store"},
	}

	for _, tt := range tests {
		got := w.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSecurityHeaders_SwaggerUsesLooserCSP(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Security-Policy"); got != swaggerCSP {
		t.Errorf("Content-Security-Policy = %q, want %q", got, swaggerCSP)
	}
}

func TestSecurityHeaders_DoesNotBlockRequest(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.POST("/data", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	req, _ := http.NewRequest("POST", "/data", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestBodySizeLimit_AllowsSmallBody(t *testing.T) {
	r := gin.New()
	r.Use(BodySizeLimit(1024)) // 1KB limit
	r.POST("/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(http.StatusOK, string(body))
	})

	smallBody := strings.Repeat("a", 100)
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(smallBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != smallBody {
		t.Errorf("expected body %q, got %q", smallBody, w.Body.String())
	}
}

func TestBodySizeLimit_RejectsLargeBody(t *testing.T) {
	r := gin.New()
	r.Use(BodySizeLimit(100)) // 100 byte limit
	r.POST("/upload", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	largeBody := strings.Repeat("a", 200)
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(largeBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestBodySizeLimit_AllowsUnderLimit(t *testing.T) {
	limit := int64(50)
	r := gin.New()
	r.Use(BodySizeLimit(limit))
	r.POST("/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(http.StatusOK, string(body))
	})

	// Body smaller than limit should succeed
	underBody := strings.Repeat("b", int(limit)-1)
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(underBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != underBody {
		t.Errorf("expected body of length %d, got %d", len(underBody), len(w.Body.String()))
	}
}

func TestBodySizeLimit_AllowsGETWithoutBody(t *testing.T) {
	r := gin.New()
	r.Use(BodySizeLimit(100))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequestTimeout_SetsContextDeadline(t *testing.T) {
	timeout := 5 * time.Second
	r := gin.New()
	r.Use(RequestTimeout(timeout))
	r.GET("/test", func(c *gin.Context) {
		deadline, ok := c.Request.Context().Deadline()
		if !ok {
			c.String(http.StatusInternalServerError, "no deadline set")
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > timeout {
			c.String(http.StatusInternalServerError, "unexpected deadline")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestRequestTimeout_DifferentDurations(t *testing.T) {
	tests := []time.Duration{
		1 * time.Second,
		10 * time.Second,
		30 * time.Second,
	}

	for _, timeout := range tests {
		t.Run(timeout.String(), func(t *testing.T) {
			r := gin.New()
			r.Use(RequestTimeout(timeout))
			r.GET("/test", func(c *gin.Context) {
				deadline, ok := c.Request.Context().Deadline()
				if !ok {
					t.Error("no deadline set")
					return
				}
				remaining := time.Until(deadline)
				if remaining > timeout {
					t.Errorf("remaining %v exceeds timeout %v", remaining, timeout)
				}
				c.String(http.StatusOK, "ok")
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestMaxRequestBodySize_LargerThanUploadSize(t *testing.T) {
	// MaxRequestBodySize must be >= the largest per-endpoint upload limit
	// so the global middleware doesn't reject uploads before handlers validate.
	// handlers.MaxUploadSize is 5MB; the global limit must be at least that.
	const handlerMaxUploadSize = 5 << 20 // mirrors handlers.MaxUploadSize
	if MaxRequestBodySize < handlerMaxUploadSize {
		t.Errorf("MaxRequestBodySize (%d) must be >= handler MaxUploadSize (%d)",
			MaxRequestBodySize, handlerMaxUploadSize)
	}
}

func TestLimitedReader_Close(t *testing.T) {
	// Test with a closeable reader
	body := io.NopCloser(strings.NewReader("hello"))
	lr := &limitedReader{reader: body, remaining: 100}
	if err := lr.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestRequestTooLargeError_Error(t *testing.T) {
	err := &requestTooLargeError{}
	if err.Error() != "request body too large" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}
