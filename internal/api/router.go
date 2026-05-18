// Package api wires the HTTP routes for the backtester service.
package api

import (
	"encoding/json"
	"net/http"

	"MFBasketBacktester/config"
	"MFBasketBacktester/internal/api/handlers"
	"MFBasketBacktester/internal/api/middleware"
)

// NewRouter builds the HTTP handler for the API, including routes and the
// per-IP rate-limiting middleware.
func NewRouter(cfg *config.Config) http.Handler {
	h := handlers.New()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /funds", h.SearchFunds)

	mux.HandleFunc("POST /baskets", h.CreateBasket)
	mux.HandleFunc("GET /baskets", h.ListBaskets)
	mux.HandleFunc("GET /baskets/{id}", h.GetBasket)
	mux.HandleFunc("DELETE /baskets/{id}", h.DeleteBasket)

	mux.HandleFunc("POST /backtest", h.RunBacktest)
	mux.HandleFunc("POST /summary", h.GetSummary)

	// Allow short bursts of 10 requests, then 5 sustained requests/second/IP.
	limiter := middleware.NewRateLimiter(5, 10)

	// Logging is outermost so it records every response, including the 429s
	// produced by the rate limiter. CORS sits just inside it so cross-origin
	// preflight requests are answered before rate limiting and routing.
	return middleware.Logging(middleware.CORS(limiter.Limit(mux)))
}
