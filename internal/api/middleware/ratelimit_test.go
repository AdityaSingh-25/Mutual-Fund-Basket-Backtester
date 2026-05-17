package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// okHandler is a trivial handler used as the wrapped target.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestRateLimiterAllowsBurst(t *testing.T) {
	rl := NewRateLimiter(1, 3) // burst of 3
	h := rl.Limit(okHandler)

	for i := 1; i <= 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d within burst: got status %d, want 200", i, rec.Code)
		}
	}
}

func TestRateLimiterRejectsOverBurst(t *testing.T) {
	rl := NewRateLimiter(1, 3)
	h := rl.Limit(okHandler)

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		h.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request over burst: got status %d, want 429", rec.Code)
	}
}

func TestRateLimiterIsolatesClients(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	h := rl.Limit(okHandler)

	exhaust := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.3:1111"
	h.ServeHTTP(exhaust, req)

	// A different client must still be allowed through.
	rec := httptest.NewRecorder()
	other := httptest.NewRequest(http.MethodGet, "/health", nil)
	other.RemoteAddr = "10.0.0.4:2222"
	h.ServeHTTP(rec, other)

	if rec.Code != http.StatusOK {
		t.Fatalf("second client: got status %d, want 200", rec.Code)
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(100, 1) // 100 tokens/sec replenishment
	h := rl.Limit(okHandler)

	send := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.5:3333"
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := send(); code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", code)
	}
	if code := send(); code != http.StatusTooManyRequests {
		t.Fatalf("immediate second request: got %d, want 429", code)
	}

	// After 50ms at 100 tokens/sec, ~5 tokens are available again.
	time.Sleep(50 * time.Millisecond)
	if code := send(); code != http.StatusOK {
		t.Fatalf("request after refill: got %d, want 200", code)
	}
}
