package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
)

func TestAuthMiddleware_RequireAuth_UserIDConversion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token with user_id as a number (will be float64 when parsed)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42, // This becomes float64 when parsed from JSON
		"email":   "test@example.com",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Set up a test handler that checks the userID type
	var capturedUserID interface{}
	testHandler := func(c *gin.Context) {
		capturedUserID, _ = c.Get(ctxkeys.UserID)
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

func TestAuthMiddleware_RequireAuth_RejectsTokenWithoutType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token WITHOUT type claim (must be rejected)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		// No "type" claim
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

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for token without type, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_AcceptsCookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a valid access token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	var capturedUserID interface{}
	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		capturedUserID, _ = c.Get(ctxkeys.UserID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// Use cookie instead of header
	req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenString})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for cookie auth, got %d", http.StatusOK, w.Code)
	}

	userIDUint, ok := capturedUserID.(uint)
	if !ok || userIDUint != 42 {
		t.Errorf("expected userID 42, got %v", capturedUserID)
	}
}

func TestAuthMiddleware_RequireAuth_PrefersCookieOverHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create two valid tokens with different user IDs
	cookieToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 100, // Cookie has user 100
		"email":   "cookie@example.com",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	cookieTokenString, _ := cookieToken.SignedString([]byte(jwtSecret))

	headerToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 200, // Header has user 200
		"email":   "header@example.com",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	headerTokenString, _ := headerToken.SignedString([]byte(jwtSecret))

	var capturedUserID interface{}
	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		capturedUserID, _ = c.Get(ctxkeys.UserID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: cookieTokenString})
	req.Header.Set("Authorization", "Bearer "+headerTokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Should use cookie token (user 100), not header token (user 200)
	userIDUint, ok := capturedUserID.(uint)
	if !ok || userIDUint != 100 {
		t.Errorf("expected cookie userID 100, got %v (cookie should take precedence)", capturedUserID)
	}
}

func TestAuthMiddleware_RequireAuth_FallsBackToHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	var capturedUserID interface{}
	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		capturedUserID, _ = c.Get(ctxkeys.UserID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No cookie, only header
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for header auth fallback, got %d", http.StatusOK, w.Code)
	}

	userIDUint, ok := capturedUserID.(uint)
	if !ok || userIDUint != 42 {
		t.Errorf("expected userID 42, got %v", capturedUserID)
	}
}

func TestAuthMiddleware_RequireAuth_InvalidCookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for invalid cookie token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_MalformedBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "invalid")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for malformed bearer header, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_EmptyBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for empty bearer token, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_BearerOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for bearer-only header, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_NonBearerScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware("test-secret")

	router := gin.New()
	router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for non-bearer scheme, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_MissingExpClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token WITHOUT exp claim to verify defense-in-depth check
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"email":   "test@example.com",
		"type":    "access",
		// No "exp" claim
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
		t.Errorf("expected status %d for missing exp claim, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_ZeroUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token with user_id: 0
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 0,
		"email":   "test@example.com",
		"type":    "access",
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
		t.Errorf("expected status %d for zero user_id, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_NegativeUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token with user_id: -1
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": -1,
		"email":   "test@example.com",
		"type":    "access",
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
		t.Errorf("expected status %d for negative user_id, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_MissingUserIDClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret"
	middleware := NewAuthMiddleware(jwtSecret)

	// Create a token WITHOUT user_id claim
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": "test@example.com",
		"type":  "access",
		"exp":   time.Now().Add(time.Hour).Unix(),
		// No "user_id" claim
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
		t.Errorf("expected status %d for missing user_id claim, got %d", http.StatusUnauthorized, w.Code)
	}
}
