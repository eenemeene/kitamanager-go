package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	// First 3 requests should be allowed
	for i := range 3 {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if rl.Allow("192.168.1.1") {
		t.Error("4th request should be blocked")
	}

	// Different IP should still be allowed
	if !rl.Allow("192.168.1.2") {
		t.Error("request from different IP should be allowed")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)

	if remaining := rl.Remaining("192.168.1.1"); remaining != 5 {
		t.Errorf("expected 5 remaining, got %d", remaining)
	}

	rl.Allow("192.168.1.1")
	if remaining := rl.Remaining("192.168.1.1"); remaining != 4 {
		t.Errorf("expected 4 remaining, got %d", remaining)
	}

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")
	if remaining := rl.Remaining("192.168.1.1"); remaining != 2 {
		t.Errorf("expected 2 remaining, got %d", remaining)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")

	if rl.Allow("192.168.1.1") {
		t.Error("should be blocked before reset")
	}

	rl.Reset("192.168.1.1")

	if !rl.Allow("192.168.1.1") {
		t.Error("should be allowed after reset")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	// Use a very short window for testing
	rl := NewRateLimiter(2, 50*time.Millisecond)

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")

	if rl.Allow("192.168.1.1") {
		t.Error("should be blocked within window")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("192.168.1.1") {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute)

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	// Launch 200 concurrent requests
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- rl.Allow("192.168.1.1")
		}()
	}

	wg.Wait()
	close(allowed)

	// Count allowed requests
	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	// Exactly 100 should be allowed
	if count != 100 {
		t.Errorf("expected exactly 100 allowed requests, got %d", count)
	}
}

func TestRateLimiter_MultipleIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	// Each IP should have its own limit
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, ip := range ips {
		if !rl.Allow(ip) {
			t.Errorf("first request from %s should be allowed", ip)
		}
		if !rl.Allow(ip) {
			t.Errorf("second request from %s should be allowed", ip)
		}
		if rl.Allow(ip) {
			t.Errorf("third request from %s should be blocked", ip)
		}
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, time.Minute)

	r := gin.New()
	r.POST("/login", rl.RateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First 2 requests should succeed
	for i := range 2 {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// 3rd request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(1, time.Minute)

	r := gin.New()
	r.POST("/login", rl.RateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First IP - should succeed then be blocked
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/login", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request from IP1: expected %d, got %d", http.StatusOK, w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/login", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request from IP1: expected %d, got %d", http.StatusTooManyRequests, w2.Code)
	}

	// Different IP - should succeed
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/login", nil)
	req3.RemoteAddr = "192.168.1.2:12345"
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("first request from IP2: expected %d, got %d", http.StatusOK, w3.Code)
	}
}

func TestLoginRateLimiter(t *testing.T) {
	rl := LoginRateLimiter(5)

	// Should allow 5 requests
	for i := range 5 {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed (login limiter allows 5)", i+1)
		}
	}

	// 6th should be blocked
	if rl.Allow("192.168.1.1") {
		t.Error("6th request should be blocked by login limiter")
	}
}

func TestLoginRateLimiter_DisabledWhenZero(t *testing.T) {
	rl := LoginRateLimiter(0)
	if rl != nil {
		t.Fatal("LoginRateLimiter(0) should return nil (disabled)")
	}
}

func TestLoginRateLimiter_DisabledWhenNegative(t *testing.T) {
	rl := LoginRateLimiter(-1)
	if rl != nil {
		t.Fatal("LoginRateLimiter(-1) should return nil (disabled)")
	}
}

func TestRateLimiter_ZeroRemaining(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1") // This one is blocked

	if remaining := rl.Remaining("192.168.1.1"); remaining != 0 {
		t.Errorf("expected 0 remaining, got %d", remaining)
	}
}

func TestAPIRateLimiter(t *testing.T) {
	rl := APIRateLimiter(3)
	if rl == nil {
		t.Fatal("APIRateLimiter(3) should not return nil")
	}

	for i := range 3 {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if rl.Allow("192.168.1.1") {
		t.Error("4th request should be blocked")
	}
}

func TestAPIRateLimiter_Disabled(t *testing.T) {
	rl := APIRateLimiter(0)
	if rl != nil {
		t.Error("APIRateLimiter(0) should return nil")
	}

	rl = APIRateLimiter(-1)
	if rl != nil {
		t.Error("APIRateLimiter(-1) should return nil")
	}
}

func TestRateLimitMutations_GETPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(1, time.Minute)

	r := gin.New()
	r.Use(rl.RateLimitMutations())
	r.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Multiple GET requests should all pass through (not rate-limited)
	for i := range 5 {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resource", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET request %d: expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}
}

func TestRateLimitMutations_POSTIsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, time.Minute)

	r := gin.New()
	r.Use(rl.RateLimitMutations())
	r.POST("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First 2 POST requests should succeed
	for i := range 2 {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/resource", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("POST request %d: expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// 3rd POST should be rate-limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/resource", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd POST request: expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestRateLimitMutations_MixedMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(1, time.Minute)

	r := gin.New()
	r.Use(rl.RateLimitMutations())
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
	r.GET("/api/resource", handler)
	r.POST("/api/resource", handler)
	r.PUT("/api/resource/:id", handler)
	r.DELETE("/api/resource/:id", handler)

	// POST uses up the single allowed mutation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/resource", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST: expected %d, got %d", http.StatusOK, w.Code)
	}

	// PUT should be blocked (same IP, mutation limit exhausted)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/resource/1", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("PUT: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	// GET should still pass through
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/resource", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET after limit: expected %d, got %d", http.StatusOK, w.Code)
	}
}
