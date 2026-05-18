package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
)

// CreateBasket handles POST /baskets — it creates a named basket along with
// its weighted fund allocations.
func (h *Handlers) CreateBasket(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBasketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "basket name is required")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "basket must contain at least one fund")
		return
	}
	seen := make(map[int]bool, len(req.Items))
	for _, item := range req.Items {
		if item.FundID <= 0 {
			writeError(w, http.StatusBadRequest, "each item needs a valid fund_id")
			return
		}
		if item.Weight <= 0 {
			writeError(w, http.StatusBadRequest, "each item weight must be positive")
			return
		}
		if seen[item.FundID] {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("fund %d appears more than once in the basket", item.FundID))
			return
		}
		seen[item.FundID] = true

		exists, err := db.FundExists(item.FundID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not validate funds")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("fund %d does not exist", item.FundID))
			return
		}
	}

	basketID, err := db.InsertBasket(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create basket")
		return
	}

	for _, item := range req.Items {
		if err := db.InsertBasketItem(basketID, item.FundID, item.Weight); err != nil {
			writeError(w, http.StatusInternalServerError, "could not add fund to basket")
			return
		}
	}

	basket, err := db.GetBasket(basketID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "basket created but could not be loaded")
		return
	}
	basket.Items, _ = db.GetBasketItems(basketID)

	writeJSON(w, http.StatusCreated, basket)
}

// ListBaskets handles GET /baskets — it returns all baskets, newest first.
func (h *Handlers) ListBaskets(w http.ResponseWriter, r *http.Request) {
	baskets, err := db.ListBaskets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list baskets")
		return
	}
	if baskets == nil {
		baskets = []models.Basket{}
	}

	writeJSON(w, http.StatusOK, baskets)
}

// DeleteBasket handles DELETE /baskets/{id} — it removes a basket and its funds.
func (h *Handlers) DeleteBasket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid basket id")
		return
	}

	affected, err := db.DeleteBasket(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete basket")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "basket not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetBasket handles GET /baskets/{id} — it returns a basket with its funds.
func (h *Handlers) GetBasket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid basket id")
		return
	}

	basket, err := db.GetBasket(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "basket not found")
		return
	}

	basket.Items, err = db.GetBasketItems(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load basket funds")
		return
	}

	writeJSON(w, http.StatusOK, basket)
}
