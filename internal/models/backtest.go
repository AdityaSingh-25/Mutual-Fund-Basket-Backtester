package models

import "time"

type NAVRecord struct {
	Date time.Time
	NAV  float64
}

// SeriesPoint is one dated portfolio value on the backtest timeline.
type SeriesPoint struct {
	Date  string  `json:"date"` // YYYY-MM-DD
	Value float64 `json:"value"`
}

// BacktestResult holds the computed performance of a basket backtest.
//
// For a lump-sum backtest TotalInvested equals the amount invested; for a SIP
// it is the sum of every monthly contribution. XIRR is the most meaningful
// return measure for a SIP, since CAGR treats all money as invested upfront.
//
// Benchmark, when present, is the same backtest run against a single
// benchmark fund over the identical date range, amount and mode.
type BacktestResult struct {
	Mode          string          `json:"mode"`      // "lumpsum" or "sip"
	Rebalance     string          `json:"rebalance"` // "none", "monthly", "quarterly" or "yearly"
	CAGR          float64         `json:"cagr"`
	XIRR          float64         `json:"xirr"`
	Drawdown      float64         `json:"drawdown"`
	TotalInvested float64         `json:"total_invested"`
	FinalValue    float64         `json:"final_value"`
	Series        []SeriesPoint   `json:"series,omitempty"`
	Benchmark     *BacktestResult `json:"benchmark,omitempty"`
}
