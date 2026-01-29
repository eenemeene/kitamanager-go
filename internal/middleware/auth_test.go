package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware_RequireAuth_UserIDConversion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token with user_id as a number (will be float64 when parsed)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42, // This becomes float64 when parsed from JSON
		"email":   "test@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Set up a test handler that checks the userID type
	var capturedUserID interface{}
	testHandler := func(c *gin.Context) {
		capturedUserID, _ = c.Get("userID")
		c.Status(http.StatusOK)
	}

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// The critical test: userID should be uint, not float64
	userIDUint, ok := capturedUserID.(uint)
	if !ok {
		t.Errorf("userID should be uint, got %T", capturedUserID)
	}

	if userIDUint != 42 {
		t.Errorf("expected userID 42, got %d", userIDUint)
	}
}

func TestAuthMiddleware_RequireAuth_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create an expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"exp":     time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
	})
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Sign with one secret, verify with another
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("secret-one"))

	middleware := NewAuthMiddleware("secret-two")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_RejectsRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a refresh token (should be rejected)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"type":    "refresh", // This should be rejected
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for refresh token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_AcceptsAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create an access token with explicit type
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"type":    "access", // Explicit access type
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for access token, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_AcceptsLegacyTokenWithoutType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token WITHOUT type claim (backwards compatibility)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		// No "type" claim - should still be accepted for backwards compatibility
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for legacy token without type, got %d", http.StatusOK, w.Code)
	}
}
