package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter stores per-IP rate limiters
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiterStore manages per-IP limiters with periodic cleanup
type rateLimiterStore struct {
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
	rps      rate.Limit
	burst    int
}

func newRateLimiterStore(rps float64, burst int) *rateLimiterStore {
	store := &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.RLock()
	entry, exists := s.limiters[ip]
	s.mu.RUnlock()

	if exists {
		s.mu.Lock()
		entry.lastSeen = time.Now()
		s.mu.Unlock()
		return entry.limiter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if entry, exists := s.limiters[ip]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(s.rps, s.burst)
	s.limiters[ip] = &ipLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

// cleanup removes stale limiters every 5 minutes
func (s *rateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for ip, entry := range s.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(s.limiters, ip)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimit returns a Gin middleware that rate-limits requests per IP
// rps is requests per second, burst is the maximum burst size
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	store := newRateLimiterStore(rps, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"message": "Too many requests"},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PublicRateLimit returns a rate limiter preset for public endpoints: 100 req/min per IP
func PublicRateLimit() gin.HandlerFunc {
	return RateLimit(100.0/60.0, 20) // ~1.67 rps, burst 20
}

// LoginRateLimit returns a rate limiter preset for login endpoints: 5 req/min per IP
func LoginRateLimit() gin.HandlerFunc {
	return RateLimit(5.0/60.0, 5) // ~0.083 rps, burst 5
}

// FormSubmitRateLimit returns a rate limiter preset for form submission: 3 req/min per IP
func FormSubmitRateLimit() gin.HandlerFunc {
	return RateLimit(3.0/60.0, 3) // 0.05 rps, burst 3
}
