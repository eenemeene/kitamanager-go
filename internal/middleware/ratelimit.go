package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/problem"
)

// TODO: This in-memory rate limiter is only suitable for single-instance deployments.
// For multi-instance/distributed deployments, replace with a Redis-backed solution
// such as github.com/ulule/limiter/v3 with the Redis driver.

// RateLimiter provides IP-based rate limiting
type RateLimiter struct {
	requests map[string]*requestInfo
	mu       sync.RWMutex
	limit    int           // Maximum requests allowed
	window   time.Duration // Time window for rate limiting
	cleanup  time.Duration // Interval to clean up old entries
	stop     chan struct{} // Signal to stop the cleanup goroutine
}

type requestInfo struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter creates a new rate limiter
// limit: maximum number of requests per window
// window: time window duration (e.g., 1 minute)
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*requestInfo),
		limit:    limit,
		window:   window,
		cleanup:  window * 2,
		stop:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes expired entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, info := range rl.requests {
				if now.After(info.windowEnd) {
					delete(rl.requests, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Stop terminates the cleanup goroutine. Call during graceful shutdown.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[ip]

	if !exists || now.After(info.windowEnd) {
		// New window
		rl.requests[ip] = &requestInfo{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true
	}

	// Within existing window
	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

// Remaining returns the number of requests remaining for an IP
func (rl *RateLimiter) Remaining(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	info, exists := rl.requests[ip]

	if !exists || now.After(info.windowEnd) {
		return rl.limit
	}

	remaining := rl.limit - info.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset resets the rate limit for a specific IP (useful for testing)
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requests, ip)
}

// RateLimit returns a Gin middleware that applies rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.Allow(ip) {
			problem.Write(c, http.StatusTooManyRequests, "rate_limit_exceeded", "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRateLimiter creates a rate limiter specifically for login endpoints.
// If limit is 0 or negative, returns nil (disabled — useful for dev/testing).
// In production, configure a positive value to prevent brute-force attacks.
func LoginRateLimiter(limit int) *RateLimiter {
	if limit <= 0 {
		return nil
	}
	return NewRateLimiter(limit, time.Minute)
}

// APIRateLimiter creates a rate limiter for API mutation endpoints.
// If limit is 0 or negative, returns nil (disabled).
func APIRateLimiter(limit int) *RateLimiter {
	if limit <= 0 {
		return nil
	}
	return NewRateLimiter(limit, time.Minute)
}

// RateLimitMutations returns a Gin middleware that rate-limits mutating requests
// (POST, PUT, PATCH, DELETE) while allowing read-only methods (GET, HEAD, OPTIONS) through.
func (rl *RateLimiter) RateLimitMutations() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		ip := c.ClientIP()
		if !rl.Allow(ip) {
			problem.Write(c, http.StatusTooManyRequests, "rate_limit_exceeded", "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}
