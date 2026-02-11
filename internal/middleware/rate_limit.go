package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// IPRateLimiter implements per-IP rate limiting using token bucket algorithm
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new rate limiter with specified requests per second and burst size
func NewIPRateLimiter(reqsPerSecond float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(reqsPerSecond),
		burst:    burst,
	}
}

// GetLimiter returns the rate limiter for a given IP address, creating one if needed
func (rl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// Middleware returns an HTTP middleware that applies rate limiting per IP address
func (rl *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
