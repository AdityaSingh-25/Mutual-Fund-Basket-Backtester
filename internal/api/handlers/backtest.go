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

// parseMode maps the request mode to a SIP flag, defaulting to lump sum.
func parseMode(mode string) (sip bool, err error) {
	switch mode {
	case "", "lumpsum":
		return false, nil
	case "sip":
		return true, nil
	default:
		return false, fmt.Errorf("invalid mode %q, expected \"lumpsum\" or \"sip\"", mode)
	}
}

// cacheKey builds a deterministic Redis key for a backtest request.
func cacheKey(req models.BacktestRequest, sip bool) string {
	mode := "lumpsum"
	if sip {
		mode = "sip"
	}
	return fmt.Sprintf("backtest:%d:%s:%s:%s:%.2f",
		req.BasketID, mode, req.StartDate, req.EndDate, req.Amount)
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

	sip, err := parseMode(req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	key := cacheKey(req, sip)

	if cached, err := cache.GetBacktestResult(key); err == nil && cached != nil {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, http.StatusOK, cached)
		return
	}

	if err := ensureBasketHistory(req.BasketID); err != nil {
		writeError(w, http.StatusBadGateway, "could not load fund history: "+err.Error())
		return
	}

	result, err := backtest.Run(req.BasketID, req.StartDate, req.EndDate, req.Amount, sip)
	if err != nil {
		writeBacktestError(w, req.BasketID, err)
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
