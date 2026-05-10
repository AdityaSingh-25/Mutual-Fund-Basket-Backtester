package models

import "time"

type NAVRecord struct {
	Date time.Time
	NAV  float64
}

type BacktestResult struct {
	CAGR       float64 `json:"cagr"`
	XIRR       float64 `json:"xirr"`
	Drawdown   float64 `json:"drawdown"`
	FinalValue float64 `json:"final_value"`
}
