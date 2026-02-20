package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const testJWTSecret = "test-secret-key-for-csrf"

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

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

	middleware := NewCSRFMiddleware(testJWTSecret)
	accessToken := "test-access-token-jwt"
	csrfToken := ComputeCSRFToken(accessToken, testJWTSecret)

	unsafeMethods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			router := gin.New()
			router.Handle(method, "/test", middleware.ValidateCSRF(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
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

func TestCSRFMiddleware_UnsafeMethods_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	// No CSRF header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for missing CSRF header, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_UnsafeMethods_WrongToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	req.Header.Set(CSRFHeaderName, "wrong-csrf-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for wrong CSRF token, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_UnsafeMethods_TokenFromDifferentSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	// Compute CSRF for a different access token
	csrfForOtherSession := ComputeCSRFToken("other-access-token", testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "my-access-token"})
	req.Header.Set(CSRFHeaderName, csrfForOtherSession)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for CSRF token from different session, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_SkipForAPIClients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

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

	middleware := NewCSRFMiddleware(testJWTSecret)

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

func TestComputeCSRFToken_Deterministic(t *testing.T) {
	token1 := ComputeCSRFToken("access-token", "secret")
	token2 := ComputeCSRFToken("access-token", "secret")
	if token1 != token2 {
		t.Error("ComputeCSRFToken should be deterministic for same inputs")
	}
}

func TestComputeCSRFToken_DifferentInputs(t *testing.T) {
	token1 := ComputeCSRFToken("access-token-1", "secret")
	token2 := ComputeCSRFToken("access-token-2", "secret")
	if token1 == token2 {
		t.Error("ComputeCSRFToken should produce different tokens for different access tokens")
	}
}

func TestComputeCSRFToken_DifferentSecrets(t *testing.T) {
	token1 := ComputeCSRFToken("access-token", "secret-1")
	token2 := ComputeCSRFToken("access-token", "secret-2")
	if token1 == token2 {
		t.Error("ComputeCSRFToken should produce different tokens for different secrets")
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

func TestCSRFMiddleware_EmptyAccessTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	// ComputeCSRFToken with empty string is deterministic
	accessToken := ""
	csrfToken := ComputeCSRFToken(accessToken, testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
	req.Header.Set(CSRFHeaderName, csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for empty access token cookie with matching CSRF, got %d: %s",
			http.StatusOK, w.Code, w.Body.String())
	}
}

func TestCSRFMiddleware_NoCookieNoAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	// No access_token cookie and no Authorization header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for no cookie and no auth header, got %d: %s",
			http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestCSRFMiddleware_ExpiredAccessTokenCookie_ValidCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewCSRFMiddleware(testJWTSecret)

	// The CSRF middleware doesn't validate the JWT itself,
	// it just checks the CSRF is derived from the cookie value.
	accessToken := "expired-access-token"
	csrfToken := ComputeCSRFToken(accessToken, testJWTSecret)

	router := gin.New()
	router.POST("/test", middleware.ValidateCSRF(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
	req.Header.Set(CSRFHeaderName, csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for expired access token with valid CSRF, got %d: %s",
			http.StatusOK, w.Code, w.Body.String())
	}
}
