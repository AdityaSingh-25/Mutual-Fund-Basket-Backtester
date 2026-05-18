package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"MFBasketBacktester/internal/backtest"
	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
)

// summaryResponse pairs the numeric backtest result with its plain-language
// summary.
type summaryResponse struct {
	Basket  string                `json:"basket"`
	Result  models.BacktestResult `json:"result"`
	Summary string                `json:"summary"`
}

func localSummary(result models.BacktestResult) string {
	return fmt.Sprintf(
		"This basket ended at ₹%.2f from ₹%.2f invested, with an XIRR of %.2f%%. "+
			"Its worst drawdown was %.2f%%, meaning the portfolio fell that much from a prior high during the period.",
		result.FinalValue,
		result.TotalInvested,
		result.XIRR,
		result.Drawdown,
	)
}

// GetSummary handles POST /summary — it runs a backtest and returns a
// plain-language summary of the result.
func (h *Handlers) GetSummary(w http.ResponseWriter, r *http.Request) {
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

	basket, err := db.GetBasket(req.BasketID)
	if err != nil {
		writeError(w, http.StatusNotFound, "basket not found")
		return
	}

	key := cacheKey(req, sip)
	var result models.BacktestResult
	if cached, err := cache.GetBacktestResult(key); err == nil && cached != nil {
		result = *cached
	} else {
		if err := ensureBasketHistory(req.BasketID); err != nil {
			writeError(w, http.StatusBadGateway, "could not load fund history: "+err.Error())
			return
		}
		result, err = backtest.Run(req.BasketID, req.StartDate, req.EndDate, req.Amount, sip)
		if err != nil {
			writeBacktestError(w, req.BasketID, err)
			return
		}
		_ = cache.SetBacktestResult(key, result)
	}

	writeJSON(w, http.StatusOK, summaryResponse{
		Basket:  basket.Name,
		Result:  result,
		Summary: localSummary(result),
	})
}
