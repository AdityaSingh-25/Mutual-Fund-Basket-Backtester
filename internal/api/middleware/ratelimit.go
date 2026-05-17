// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// bucket is a per-client token bucket.
type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter throttles requests per client IP using a token-bucket algorithm.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	rate    float64 // tokens replenished per second
	burst   float64 // maximum tokens a client may accumulate
}

// NewRateLimiter builds a limiter allowing burst requests immediately and
// rate sustained requests per second thereafter. It starts a background
// goroutine that evicts idle clients.
func NewRateLimiter(rate, burst float64) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
	go rl.cleanup()
	return rl
}

// cleanup periodically drops clients that have been idle for 10 minutes.
func (rl *RateLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		for ip, b := range rl.clients {
			if time.Since(b.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow reports whether a request from ip may proceed, consuming a token.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.clients[ip]
	if !ok {
		rl.clients[ip] = &bucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	// Replenish tokens based on elapsed time, capped at burst.
	b.tokens += now.Sub(b.lastSeen).Seconds() * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Limit wraps next, rejecting requests that exceed the configured rate.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded, slow down",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
