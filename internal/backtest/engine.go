// Package backtest runs a weighted mutual-fund basket against historical NAV
// data and reports portfolio performance metrics.
package backtest

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"MFBasketBacktester/internal/compute"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
)

const dateLayout = "2006-01-02"

// Run simulates a lump-sum investment of amount into the given basket at
// startDate and holds it until endDate, returning CAGR, XIRR, max drawdown
// and the final portfolio value.
//
// Each fund's units are bought at its first available NAV on or after the
// common start date; portfolio value on any later date is the sum of each
// fund's units valued at its most recent NAV (forward-filled).
func Run(basketID int, startDate, endDate string, amount float64) (models.BacktestResult, error) {
	var result models.BacktestResult

	if amount <= 0 {
		return result, errors.New("amount must be positive")
	}

	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return result, errors.New("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return result, errors.New("invalid end_date, expected YYYY-MM-DD")
	}
	if !end.After(start) {
		return result, errors.New("end_date must be after start_date")
	}

	items, err := db.GetBasketItems(basketID)
	if err != nil {
		return result, err
	}
	if len(items) == 0 {
		return result, errors.New("basket has no funds")
	}

	totalWeight := 0.0
	for _, it := range items {
		if it.Weight < 0 {
			return result, errors.New("fund weight cannot be negative")
		}
		totalWeight += it.Weight
	}
	if totalWeight <= 0 {
		return result, errors.New("basket weights sum to zero")
	}

	// Load NAV history for every fund and find the common start date —
	// the latest date on which all funds first have data.
	type fundSeries struct {
		records []models.NAVRecord
		weight  float64 // normalised to sum to 1 across the basket
	}
	series := make([]fundSeries, 0, len(items))
	var effectiveStart time.Time

	for _, it := range items {
		records, err := db.GetNAVHistory(it.FundID, startDate, endDate)
		if err != nil {
			return result, err
		}
		if len(records) == 0 {
			return result, fmt.Errorf("fund %d has no NAV data in the requested range", it.FundID)
		}
		series = append(series, fundSeries{
			records: records,
			weight:  it.Weight / totalWeight,
		})
		if first := records[0].Date; first.After(effectiveStart) {
			effectiveStart = first
		}
	}

	// Reject funds whose data ends before the common start date.
	for i, s := range series {
		if last := s.records[len(s.records)-1].Date; last.Before(effectiveStart) {
			return result, fmt.Errorf("fund %d has no NAV data after %s",
				items[i].FundID, effectiveStart.Format(dateLayout))
		}
	}

	// Union of all observation dates on or after the common start.
	dateSet := make(map[time.Time]struct{})
	for _, s := range series {
		for _, r := range s.records {
			if !r.Date.Before(effectiveStart) {
				dateSet[r.Date] = struct{}{}
			}
		}
	}
	dates := make([]time.Time, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	if len(dates) < 2 {
		return result, errors.New("insufficient NAV data points to run a backtest")
	}

	// Walk the timeline, forward-filling each fund's NAV, and build the
	// portfolio value series. Units are fixed at the common start date.
	pointers := make([]int, len(series))
	units := make([]float64, len(series))
	values := make([]float64, 0, len(dates))

	for di, date := range dates {
		total := 0.0
		for si := range series {
			recs := series[si].records
			p := pointers[si]
			for p+1 < len(recs) && !recs[p+1].Date.After(date) {
				p++
			}
			pointers[si] = p

			nav := recs[p].NAV
			if di == 0 {
				if nav <= 0 {
					return result, fmt.Errorf("fund %d has a non-positive NAV at start",
						items[si].FundID)
				}
				units[si] = (amount * series[si].weight) / nav
			}
			total += units[si] * nav
		}
		values = append(values, total)
	}

	finalValue := values[len(values)-1]
	years := dates[len(dates)-1].Sub(dates[0]).Hours() / 24 / 365

	result.CAGR = round2(compute.CAGR(amount, finalValue, years))
	result.Drawdown = round2(compute.MaxDrawdown(values))
	result.XIRR = round2(compute.XIRR([]compute.Cashflow{
		{Date: dates[0], Amount: -amount},
		{Date: dates[len(dates)-1], Amount: finalValue},
	}))
	result.FinalValue = round2(finalValue)

	return result, nil
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
