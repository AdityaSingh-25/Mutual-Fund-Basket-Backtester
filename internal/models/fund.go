package models

import "time"

// Fund is a mutual fund scheme as stored in the funds table.
type Fund struct {
	ID         int       `json:"id"`
	SchemeCode int64     `json:"scheme_code"`
	SchemeName string    `json:"scheme_name"`
	FundHouse  string    `json:"fund_house,omitempty"`
	SchemeType string    `json:"scheme_type,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// BasketItem is a single fund allocation within a basket.
type BasketItem struct {
	ID       int     `json:"id"`
	BasketID int     `json:"basket_id"`
	FundID   int     `json:"fund_id"`
	Weight   float64 `json:"weight"`
}
