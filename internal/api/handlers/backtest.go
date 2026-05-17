package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"MFBasketBacktester/internal/backtest"
	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/ingestion"
	"MFBasketBacktester/internal/models"
)

// cacheKey builds a deterministic Redis key for a backtest request.
func cacheKey(req models.BacktestRequest) string {
	return fmt.Sprintf("backtest:%d:%s:%s:%.2f",
		req.BasketID, req.StartDate, req.EndDate, req.Amount)
}

// ensureBasketHistory backfills full NAV history for every fund in a basket
// that lacks it, so the backtest engine has enough data points to run.
func ensureBasketHistory(basketID int) error {
	items, err := db.GetBasketItems(basketID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := ingestion.EnsureHistory(item.FundID); err != nil {
			return err
		}
	}
	return nil
}

// RunBacktest handles POST /backtest — it runs (or serves a cached) backtest
// for a basket over a date range.
func (h *Handlers) RunBacktest(w http.ResponseWriter, r *http.Request) {
	var req models.BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.BasketID <= 0 {
		writeError(w, http.StatusBadRequest, "valid basket_id is required")
		return
	}

	key := cacheKey(req)

	if cached, err := cache.GetBacktestResult(key); err == nil && cached != nil {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, http.StatusOK, cached)
		return
	}

	if err := ensureBasketHistory(req.BasketID); err != nil {
		writeError(w, http.StatusBadGateway, "could not load fund history: "+err.Error())
		return
	}

	result, err := backtest.Run(req.BasketID, req.StartDate, req.EndDate, req.Amount)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := cache.SetBacktestResult(key, result); err != nil {
		// Caching is best-effort; a failure here must not fail the request.
		w.Header().Set("X-Cache", "SKIP")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	writeJSON(w, http.StatusOK, result)
}
