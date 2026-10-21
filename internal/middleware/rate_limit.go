package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	// requests stores the timestamps of requests for each key
	requests map[string][]time.Time
	// mutex protects access to the requests map
	mutex sync.Mutex
	// maxRequests is the maximum number of requests allowed
	maxRequests int
	// window is the time window for rate limiting
	window time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}
}

// Check checks if the request is allowed
func (rl *RateLimiter) Check(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Clean old requests
	now := time.Now()
	oldRequests := make([]time.Time, 0)
	for _, t := range rl.requests[key] {
		if now.Sub(t) < rl.window {
			oldRequests = append(oldRequests, t)
		}
	}
	rl.requests[key] = oldRequests

	// Check if we've exceeded the limit
	if len(oldRequests) >= rl.maxRequests {
		return false
	}

	// Record the current request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// RateLimitMiddleware returns a middleware that rate limits requests
func RateLimitMiddleware(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(maxRequests, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract a key from the request (e.g., IP address or user ID)
			key := r.RemoteAddr
			if user, ok := UserFromContext(r.Context()); ok {
				key = user.Username + "|" + r.RemoteAddr
			}

			// Check if the request is allowed
			if !limiter.Check(key) {
				writeAPIError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GitRateLimitMiddleware returns a middleware that rate limits Git operations
func GitRateLimitMiddleware() func(http.Handler) http.Handler {
	// Git operations are more resource-intensive, so we use a stricter limit
	return RateLimitMiddleware(10, time.Minute)
}

// APIRateLimitMiddleware returns a middleware that rate limits API requests
func APIRateLimitMiddleware() func(http.Handler) http.Handler {
	// API requests are less resource-intensive, but we still want to limit them
	return RateLimitMiddleware(100, time.Minute)
}
