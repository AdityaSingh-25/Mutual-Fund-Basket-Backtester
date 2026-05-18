package ingestion

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MFBasketBacktester/internal/db"
)

// mfapiURL serves the full NAV history for a single scheme. AMFI's own feed
// only exposes the latest NAV, so historical backtests rely on this source.
const mfapiURL = "https://api.mfapi.in/mf/%d"

// mfapiDateLayout is the date format used by api.mfapi.in (e.g. "16-05-2026").
const mfapiDateLayout = "02-01-2006"

type mfapiResponse struct {
	Meta struct {
		SchemeCode int64  `json:"scheme_code"`
		SchemeName string `json:"scheme_name"`
	} `json:"meta"`
	Data []struct {
		Date string `json:"date"`
		NAV  string `json:"nav"`
	} `json:"data"`
	Status string `json:"status"`
}

// fetchHistory performs the mfapi GET, retrying up to three times with a
// linear backoff on a network error or a 5xx response. A 4xx response is
// returned to the caller without a retry.
func fetchHistory(schemeCode int64) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf(mfapiURL, schemeCode)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := client.Get(url)
		switch {
		case err != nil:
			lastErr = err
		case resp.StatusCode >= 500:
			resp.Body.Close()
			lastErr = fmt.Errorf("mfapi returned status %d", resp.StatusCode)
		default:
			return resp, nil
		}
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, fmt.Errorf("mfapi request for scheme %d failed after 3 attempts: %w",
		schemeCode, lastErr)
}

// FetchAndStoreHistory pulls the complete NAV history for a scheme from
// api.mfapi.in and stores it against the given fund. It returns the number of
// new NAV rows inserted (existing dates are left untouched).
func FetchAndStoreHistory(fundID int, schemeCode int64) (int, error) {
	resp, err := fetchHistory(schemeCode)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("scheme %d not found on mfapi", schemeCode)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("mfapi returned status %d for scheme %d", resp.StatusCode, schemeCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var parsed mfapiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("could not parse mfapi response for scheme %d: %w", schemeCode, err)
	}
	if len(parsed.Data) == 0 {
		return 0, fmt.Errorf("mfapi returned no NAV history for scheme %d", schemeCode)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO nav (fund_id, nav, date)
		VALUES ($1, $2, $3)
		ON CONFLICT (fund_id, date) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, rec := range parsed.Data {
		navValue, err := strconv.ParseFloat(strings.TrimSpace(rec.NAV), 64)
		if err != nil || navValue <= 0 {
			continue
		}

		date, err := time.Parse(mfapiDateLayout, strings.TrimSpace(rec.Date))
		if err != nil {
			continue
		}

		res, err := stmt.Exec(fundID, navValue, date)
		if err != nil {
			return inserted, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}

	return inserted, nil
}

// EnsureHistory guarantees a fund has enough NAV data points for a backtest,
// backfilling its full history from mfapi if fewer than two dates are stored.
func EnsureHistory(fundID int) error {
	count, err := db.CountNAVDates(fundID)
	if err != nil {
		return err
	}
	if count >= 2 {
		return nil
	}

	fund, err := db.GetFund(fundID)
	if err != nil {
		return fmt.Errorf("fund %d not found: %w", fundID, err)
	}

	inserted, err := FetchAndStoreHistory(fundID, fund.SchemeCode)
	if err != nil {
		return err
	}

	log.Printf("backfilled %d NAV records for fund %d (scheme %d)",
		inserted, fundID, fund.SchemeCode)
	return nil
}
