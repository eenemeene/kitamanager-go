package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			router := gin.New()
			router.Handle(method, "/test", middleware.ValidateCSRF(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d for safe method %s, got %d", http.StatusOK, method, w.Code)
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethods_WithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()
	csrfToken := "test-csrf-token-12345"

	unsafeMethods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			router := gin.New()
			router.Handle(method, "/test", middleware.ValidateCSRF(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
			req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfToken})
			req.Header.Set(CSRFHeaderName, csrfToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d for method %s with valid CSRF, got %d: %s",
					http.StatusOK, method, w.Code, w.Body.String())
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethods_MissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	// No CSRF cookie
	req.Header.Set(CSRFHeaderName, "some-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for missing CSRF cookie, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_UnsafeMethods_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf-token"})
	// No CSRF header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for missing CSRF header, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_UnsafeMethods_MismatchedTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "cookie-token"})
	req.Header.Set(CSRFHeaderName, "different-header-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for mismatched CSRF tokens, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_SkipForAPIClients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	// No access_token cookie, but has Authorization header (API client)
	req.Header.Set("Authorization", "Bearer some-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should pass because API clients using header auth skip CSRF
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for API client with header auth, got %d: %s",
			http.StatusOK, w.Code, w.Body.String())
	}
}

func TestCSRFMiddleware_RequireCSRFForCookieAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware()

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	// Has access_token cookie (cookie auth), no CSRF
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail because cookie auth requires CSRF
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for cookie auth without CSRF, got %d",
			http.StatusForbidden, w.Code)
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"hello", "hello", true},
		{"hello", "world", false},
		{"hello", "hell", false},
		{"", "", true},
		{"a", "", false},
		{"abc123", "abc123", true},
		{"abc123", "abc124", false},
	}

	for _, tc := range tests {
		result := secureCompare(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("secureCompare(%q, %q) = %v, expected %v", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestIsSafeMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"POST", false},
		{"PUT", false},
		{"PATCH", false},
		{"DELETE", false},
	}

	for _, tc := range tests {
		result := isSafeMethod(tc.method)
		if result != tc.expected {
			t.Errorf("isSafeMethod(%q) = %v, expected %v", tc.method, result, tc.expected)
		}
	}
}
