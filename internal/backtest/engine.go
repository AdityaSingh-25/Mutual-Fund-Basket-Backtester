// Package backtest runs a weighted mutual-fund basket against historical NAV
// data and reports portfolio performance metrics.
package backtest

import (
	"fmt"
	"math"
	"sort"
	"time"

	"MFBasketBacktester/internal/compute"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
)

const dateLayout = "2006-01-02"

// ValidationError marks a backtest failure caused by invalid input — bad
// dates, an empty basket, insufficient NAV data — as distinct from an
// internal error such as a database failure. Callers use errors.As to map it
// to an HTTP 400 rather than a 500.
type ValidationError struct{ Err error }

func (e ValidationError) Error() string { return e.Err.Error() }
func (e ValidationError) Unwrap() error { return e.Err }

// invalid builds a ValidationError from a formatted message.
func invalid(format string, args ...any) error {
	return ValidationError{Err: fmt.Errorf(format, args...)}
}

// rebalanceMonths maps a rebalance period to its length in months. It returns
// 0 for "none" or any unrecognised value, meaning no rebalancing.
func rebalanceMonths(period string) int {
	switch period {
	case "monthly":
		return 1
	case "quarterly":
		return 3
	case "yearly":
		return 12
	default:
		return 0
	}
}

// FundInput is one fund's basket allocation paired with its NAV history.
// Records must be sorted by date ascending.
type FundInput struct {
	Label   string // human-readable identifier used in error messages
	Weight  float64
	Records []models.NAVRecord
}

// Run simulates an investment of amount into the given basket between
// startDate and endDate (both YYYY-MM-DD), loading NAV data from the database.
// When sip is true, amount is invested every month; otherwise it is a single
// lump sum at the start.
//
// Input problems are returned as ValidationError; database failures are
// returned as plain errors.
func Run(basketID int, startDate, endDate string, amount float64, sip bool, rebalance string) (models.BacktestResult, error) {
	var result models.BacktestResult

	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return result, invalid("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return result, invalid("invalid end_date, expected YYYY-MM-DD")
	}
	if !end.After(start) {
		return result, invalid("end_date must be after start_date")
	}

	items, err := db.GetBasketItems(basketID)
	if err != nil {
		return result, err
	}
	if len(items) == 0 {
		return result, invalid("basket has no funds")
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

	return Simulate(funds, amount, sip, rebalance)
}

// RunFund backtests a single fund as a 100%-weight portfolio, loading its NAV
// data from the database. It is used to produce benchmark comparisons over
// the same date range, amount and mode as a basket backtest.
func RunFund(fundID int, startDate, endDate string, amount float64, sip bool, rebalance string) (models.BacktestResult, error) {
	var result models.BacktestResult

	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return result, invalid("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return result, invalid("invalid end_date, expected YYYY-MM-DD")
	}
	if !end.After(start) {
		return result, invalid("end_date must be after start_date")
	}

	records, err := db.GetNAVHistory(fundID, startDate, endDate)
	if err != nil {
		return result, err
	}

	return Simulate([]FundInput{{
		Label:   fmt.Sprintf("benchmark fund %d", fundID),
		Weight:  1,
		Records: records,
	}}, amount, sip, rebalance)
}

// Simulate is the pure, database-free core of a backtest. It splits each
// contribution across funds by normalised weight and buys units at that day's
// NAVs; portfolio value on any later date is the sum of accumulated units
// valued at each fund's most recent NAV (forward-filled).
//
// When sip is false a single contribution of amount is made on the common
// start date. When sip is true, amount is contributed on the start date and
// then monthly thereafter. rebalance ("monthly", "quarterly" or "yearly")
// periodically resets holdings to the target weights; any other value
// disables it. All errors it returns are ValidationError.
func Simulate(funds []FundInput, amount float64, sip bool, rebalance string) (models.BacktestResult, error) {
	var result models.BacktestResult

	if amount <= 0 {
		return result, invalid("amount must be positive")
	}
	if len(funds) == 0 {
		return result, invalid("no funds provided")
	}

	totalWeight := 0.0
	for _, f := range funds {
		if f.Weight < 0 {
			return result, invalid("fund weight cannot be negative")
		}
		totalWeight += f.Weight
	}
	if totalWeight <= 0 {
		return result, invalid("basket weights sum to zero")
	}

	// Find the common start date — the latest date on which all funds first
	// have data — and reject funds with no usable data.
	var effectiveStart time.Time
	for _, f := range funds {
		if len(f.Records) == 0 {
			return result, invalid("%s has no NAV data in the requested range", f.Label)
		}
		if first := f.Records[0].Date; first.After(effectiveStart) {
			effectiveStart = first
		}
	}
	for _, f := range funds {
		if last := f.Records[len(f.Records)-1].Date; last.Before(effectiveStart) {
			return result, invalid("%s has no NAV data after %s",
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
		return result, invalid("insufficient NAV data points to run a backtest")
	}

	// Build the contribution schedule: amount per timeline-date index. A SIP
	// contributes monthly from the start; a lump sum contributes once.
	contributions := make([]float64, len(dates))
	if sip {
		lastDate := dates[len(dates)-1]
		prevIdx := -1
		for month := 0; ; month++ {
			target := effectiveStart.AddDate(0, month, 0)
			if month > 0 && target.After(lastDate) {
				break
			}
			// First timeline date on or after the monthly anniversary.
			idx := sort.Search(len(dates), func(i int) bool {
				return !dates[i].Before(target)
			})
			if idx >= len(dates) {
				break
			}
			if idx == prevIdx { // anniversary fell in a data gap already used
				continue
			}
			prevIdx = idx
			contributions[idx] += amount
		}
	} else {
		contributions[0] = amount
	}

	// Build the rebalance schedule: timeline indices on which holdings are
	// reset to target weights. Empty when rebalancing is disabled.
	rebalanceAt := make([]bool, len(dates))
	if months := rebalanceMonths(rebalance); months > 0 {
		lastDate := dates[len(dates)-1]
		for n := months; ; n += months {
			target := effectiveStart.AddDate(0, n, 0)
			if target.After(lastDate) {
				break
			}
			idx := sort.Search(len(dates), func(i int) bool {
				return !dates[i].Before(target)
			})
			if idx >= len(dates) {
				break
			}
			rebalanceAt[idx] = true
		}
	}

	// Walk the timeline: forward-fill each fund's NAV, apply any contribution
	// due that day, then record the portfolio value.
	pointers := make([]int, len(funds))
	units := make([]float64, len(funds))
	navs := make([]float64, len(funds))
	values := make([]float64, len(dates))

	for di, date := range dates {
		for fi := range funds {
			recs := funds[fi].Records
			p := pointers[fi]
			for p+1 < len(recs) && !recs[p+1].Date.After(date) {
				p++
			}
			pointers[fi] = p
			navs[fi] = recs[p].NAV
		}

		if c := contributions[di]; c > 0 {
			for fi := range funds {
				if navs[fi] <= 0 {
					return result, invalid("%s has a non-positive NAV on %s",
						funds[fi].Label, date.Format(dateLayout))
				}
				units[fi] += (c * funds[fi].Weight / totalWeight) / navs[fi]
			}
		}

		total := 0.0
		for fi := range funds {
			total += units[fi] * navs[fi]
		}

		// Rebalancing redistributes existing holdings back to the target
		// weights. It changes each fund's unit count but not the total.
		if rebalanceAt[di] && total > 0 {
			positive := true
			for fi := range funds {
				if navs[fi] <= 0 {
					positive = false
				}
			}
			if positive {
				for fi := range funds {
					units[fi] = (total * funds[fi].Weight / totalWeight) / navs[fi]
				}
			}
		}

		values[di] = total
	}

	// Cashflows for XIRR: each contribution is money out, the final value is
	// money returned.
	var cashflows []compute.Cashflow
	totalInvested := 0.0
	for di, c := range contributions {
		if c > 0 {
			cashflows = append(cashflows, compute.Cashflow{Date: dates[di], Amount: -c})
			totalInvested += c
		}
	}
	finalValue := values[len(values)-1]
	cashflows = append(cashflows, compute.Cashflow{
		Date:   dates[len(dates)-1],
		Amount: finalValue,
	})

	years := dates[len(dates)-1].Sub(dates[0]).Hours() / 24 / 365

	series := make([]models.SeriesPoint, len(dates))
	for i := range dates {
		series[i] = models.SeriesPoint{
			Date:  dates[i].Format(dateLayout),
			Value: round2(values[i]),
		}
	}

	result.Mode = "lumpsum"
	if sip {
		result.Mode = "sip"
	}
	result.Rebalance = "none"
	if rebalanceMonths(rebalance) > 0 {
		result.Rebalance = rebalance
	}
	result.CAGR = round2(compute.CAGR(totalInvested, finalValue, years))
	result.XIRR = round2(compute.XIRR(cashflows))
	result.Drawdown = round2(compute.MaxDrawdown(values))
	result.TotalInvested = round2(totalInvested)
	result.FinalValue = round2(finalValue)
	result.Series = series

	return result, nil
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
