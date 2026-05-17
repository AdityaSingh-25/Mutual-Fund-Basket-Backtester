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

// FundInput is one fund's basket allocation paired with its NAV history.
// Records must be sorted by date ascending.
type FundInput struct {
	Label   string // human-readable identifier used in error messages
	Weight  float64
	Records []models.NAVRecord
}

// Run simulates a lump-sum investment of amount into the given basket between
// startDate and endDate (both YYYY-MM-DD), loading NAV data from the database.
func Run(basketID int, startDate, endDate string, amount float64) (models.BacktestResult, error) {
	var result models.BacktestResult

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

	funds := make([]FundInput, 0, len(items))
	for _, it := range items {
		records, err := db.GetNAVHistory(it.FundID, startDate, endDate)
		if err != nil {
			return result, err
		}
		funds = append(funds, FundInput{
			Label:   fmt.Sprintf("fund %d", it.FundID),
			Weight:  it.Weight,
			Records: records,
		})
	}

	return Simulate(funds, amount)
}

// Simulate is the pure, database-free core of a backtest. It values a lump-sum
// investment of amount, split across funds by normalised weight, from the
// common start date (the latest date on which every fund first has data)
// through the last observation, and reports CAGR, XIRR, max drawdown and the
// final portfolio value.
//
// Each fund's units are fixed at the common start date; portfolio value on any
// later date is the sum of each fund's units valued at its most recent NAV
// (forward-filled).
func Simulate(funds []FundInput, amount float64) (models.BacktestResult, error) {
	var result models.BacktestResult

	if amount <= 0 {
		return result, errors.New("amount must be positive")
	}
	if len(funds) == 0 {
		return result, errors.New("no funds provided")
	}

	totalWeight := 0.0
	for _, f := range funds {
		if f.Weight < 0 {
			return result, errors.New("fund weight cannot be negative")
		}
		totalWeight += f.Weight
	}
	if totalWeight <= 0 {
		return result, errors.New("basket weights sum to zero")
	}

	// Find the common start date — the latest date on which all funds first
	// have data — and reject funds with no usable data.
	var effectiveStart time.Time
	for _, f := range funds {
		if len(f.Records) == 0 {
			return result, fmt.Errorf("%s has no NAV data in the requested range", f.Label)
		}
		if first := f.Records[0].Date; first.After(effectiveStart) {
			effectiveStart = first
		}
	}
	for _, f := range funds {
		if last := f.Records[len(f.Records)-1].Date; last.Before(effectiveStart) {
			return result, fmt.Errorf("%s has no NAV data after %s",
				f.Label, effectiveStart.Format(dateLayout))
		}
	}

	// Union of all observation dates on or after the common start.
	dateSet := make(map[time.Time]struct{})
	for _, f := range funds {
		for _, r := range f.Records {
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
	pointers := make([]int, len(funds))
	units := make([]float64, len(funds))
	values := make([]float64, 0, len(dates))

	for di, date := range dates {
		total := 0.0
		for fi := range funds {
			recs := funds[fi].Records
			p := pointers[fi]
			for p+1 < len(recs) && !recs[p+1].Date.After(date) {
				p++
			}
			pointers[fi] = p

			nav := recs[p].NAV
			if di == 0 {
				if nav <= 0 {
					return result, fmt.Errorf("%s has a non-positive NAV at start", funds[fi].Label)
				}
				units[fi] = (amount * funds[fi].Weight / totalWeight) / nav
			}
			total += units[fi] * nav
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
