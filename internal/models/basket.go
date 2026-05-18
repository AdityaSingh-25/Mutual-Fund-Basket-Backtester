package models

import "time"

// Basket is a named, user-defined portfolio of funds.
type Basket struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	CreatedAt time.Time    `json:"created_at"`
	Items     []BasketItem `json:"items,omitempty"`
}

// CreateBasketRequest is the payload for POST /baskets.
type CreateBasketRequest struct {
	Name  string                  `json:"name"`
	Items []CreateBasketItemInput `json:"items"`
}

// CreateBasketItemInput is one allocation in a create-basket request.
type CreateBasketItemInput struct {
	FundID int     `json:"fund_id"`
	Weight float64 `json:"weight"`
}

// BacktestRequest is the payload for POST /backtest.
//
// Mode selects the investment style: "lumpsum" (default) invests Amount once
// at the start; "sip" invests Amount every month.
type BacktestRequest struct {
	BasketID  int     `json:"basket_id"`
	StartDate string  `json:"start_date"` // YYYY-MM-DD
	EndDate   string  `json:"end_date"`   // YYYY-MM-DD
	Amount    float64 `json:"amount"`     // lump-sum total, or monthly SIP amount
	Mode      string  `json:"mode"`       // "lumpsum" (default) or "sip"
}
