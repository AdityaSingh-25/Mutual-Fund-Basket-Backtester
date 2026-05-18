package handlers

import (
	"net/http"
	"strconv"

	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
)

const (
	defaultFundLimit = 20
	maxFundLimit     = 100
)

// SearchFunds handles GET /funds — it searches funds by scheme name or scheme
// code. Query params: q (search term, optional), limit (default 20, max 100),
// and offset (default 0) for pagination.
func (h *Handlers) SearchFunds(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	limit := defaultFundLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = n
	}
	if limit > maxFundLimit {
		limit = maxFundLimit
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "offset must be a non-negative integer")
			return
		}
		offset = n
	}

	funds, err := db.SearchFunds(q, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search funds")
		return
	}
	if funds == nil {
		funds = []models.Fund{}
	}

	writeJSON(w, http.StatusOK, funds)
}

// GetFund handles GET /funds/{id} — it returns a single fund by id.
func (h *Handlers) GetFund(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid fund id")
		return
	}

	fund, err := db.GetFund(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "fund not found")
		return
	}

	writeJSON(w, http.StatusOK, fund)
}
