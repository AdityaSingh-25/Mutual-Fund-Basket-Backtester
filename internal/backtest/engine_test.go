package backtest

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"MFBasketBacktester/internal/models"
)

// nav builds a NAV record from a YYYY-MM-DD date string and a value.
func nav(t *testing.T, date string, value float64) models.NAVRecord {
	t.Helper()
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		t.Fatalf("bad test date %q: %v", date, err)
	}
	return models.NAVRecord{Date: d, NAV: value}
}

func TestSimulate(t *testing.T) {
	cases := []struct {
		name          string
		funds         func(t *testing.T) []FundInput
		amount        float64
		wantCAGR      float64
		wantXIRR      float64
		wantDrawdown  float64
		wantFinal     float64
		wantSeriesLen int
	}{
		{
			// 100 units bought at NAV 100; NAV grows 50% over exactly one year.
			name: "single fund, one year, 50% growth",
			funds: func(t *testing.T) []FundInput {
				return []FundInput{{
					Label:  "fund A",
					Weight: 1,
					Records: []models.NAVRecord{
						nav(t, "2021-01-01", 100),
						nav(t, "2022-01-01", 150),
					},
				}}
			},
			amount:        10000,
			wantCAGR:      50,
			wantXIRR:      50,
			wantDrawdown:  0,
			wantFinal:     15000,
			wantSeriesLen: 2,
		},
		{
			// Weights 6 and 4 must be normalised to 60% / 40%.
			name: "two funds, unnormalised weights",
			funds: func(t *testing.T) []FundInput {
				return []FundInput{
					{Label: "fund A", Weight: 6, Records: []models.NAVRecord{
						nav(t, "2021-01-01", 100),
						nav(t, "2022-01-01", 120),
					}},
					{Label: "fund B", Weight: 4, Records: []models.NAVRecord{
						nav(t, "2021-01-01", 50),
						nav(t, "2022-01-01", 75),
					}},
				}
			},
			amount:        10000,
			wantCAGR:      32,
			wantXIRR:      32,
			wantDrawdown:  0,
			wantFinal:     13200,
			wantSeriesLen: 2,
		},
		{
			// A mid-period dip to NAV 80 is a 20% drawdown from the 100 peak.
			name: "single fund with a drawdown",
			funds: func(t *testing.T) []FundInput {
				return []FundInput{{
					Label:  "fund A",
					Weight: 1,
					Records: []models.NAVRecord{
						nav(t, "2021-01-01", 100),
						nav(t, "2021-07-01", 80),
						nav(t, "2022-01-01", 120),
					},
				}}
			},
			amount:        10000,
			wantCAGR:      20,
			wantXIRR:      20,
			wantDrawdown:  20,
			wantFinal:     12000,
			wantSeriesLen: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Simulate(tc.funds(t), tc.amount, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.CAGR != tc.wantCAGR {
				t.Errorf("CAGR = %v, want %v", got.CAGR, tc.wantCAGR)
			}
			if got.XIRR != tc.wantXIRR {
				t.Errorf("XIRR = %v, want %v", got.XIRR, tc.wantXIRR)
			}
			if got.Drawdown != tc.wantDrawdown {
				t.Errorf("Drawdown = %v, want %v", got.Drawdown, tc.wantDrawdown)
			}
			if got.FinalValue != tc.wantFinal {
				t.Errorf("FinalValue = %v, want %v", got.FinalValue, tc.wantFinal)
			}
			if got.Mode != "lumpsum" {
				t.Errorf("Mode = %q, want %q", got.Mode, "lumpsum")
			}
			if got.TotalInvested != tc.amount {
				t.Errorf("TotalInvested = %v, want %v", got.TotalInvested, tc.amount)
			}
			if len(got.Series) != tc.wantSeriesLen {
				t.Errorf("len(Series) = %d, want %d", len(got.Series), tc.wantSeriesLen)
			}
		})
	}
}

// TestSimulateClampsToCommonStart verifies the backtest begins on the latest
// date at which every fund has data, not the earliest date overall.
func TestSimulateClampsToCommonStart(t *testing.T) {
	funds := []FundInput{
		{Label: "fund A", Weight: 1, Records: []models.NAVRecord{
			nav(t, "2021-01-01", 100), // earlier history, ignored
			nav(t, "2021-07-01", 110),
			nav(t, "2022-01-01", 121),
		}},
		{Label: "fund B", Weight: 1, Records: []models.NAVRecord{
			nav(t, "2021-07-01", 200), // fund B only starts here
			nav(t, "2022-01-01", 240),
		}},
	}

	got, err := Simulate(funds, 10000, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Units fixed at 2021-07-01: A = 5000/110, B = 5000/200.
	// Final at 2022-01-01: (5000/110)*121 + (5000/200)*240 = 5500 + 6000.
	if got.FinalValue != 11500 {
		t.Errorf("FinalValue = %v, want 11500 (start clamped to 2021-07-01)", got.FinalValue)
	}
	if got.Drawdown != 0 {
		t.Errorf("Drawdown = %v, want 0", got.Drawdown)
	}
}

// TestSimulateSIP checks that a monthly SIP makes one contribution per month,
// accumulating units. With a flat NAV the investor simply gets their money
// back: zero growth and zero drawdown.
func TestSimulateSIP(t *testing.T) {
	funds := []FundInput{{
		Label:  "fund A",
		Weight: 1,
		Records: []models.NAVRecord{
			nav(t, "2021-01-01", 100),
			nav(t, "2021-02-01", 100),
			nav(t, "2021-03-01", 100),
			nav(t, "2021-04-01", 100),
		},
	}}

	got, err := Simulate(funds, 1000, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Four monthly contributions of 1000, each buying 10 units at NAV 100.
	if got.Mode != "sip" {
		t.Errorf("Mode = %q, want %q", got.Mode, "sip")
	}
	if got.TotalInvested != 4000 {
		t.Errorf("TotalInvested = %v, want 4000", got.TotalInvested)
	}
	if got.FinalValue != 4000 {
		t.Errorf("FinalValue = %v, want 4000", got.FinalValue)
	}
	if got.CAGR != 0 {
		t.Errorf("CAGR = %v, want 0", got.CAGR)
	}
	if got.Drawdown != 0 {
		t.Errorf("Drawdown = %v, want 0", got.Drawdown)
	}
	if math.Abs(got.XIRR) > 0.5 {
		t.Errorf("XIRR = %v, want approximately 0", got.XIRR)
	}
	if len(got.Series) != 4 {
		t.Errorf("len(Series) = %d, want 4", len(got.Series))
	}
}

func TestSimulateErrors(t *testing.T) {
	oneFund := func(weight float64, recs ...models.NAVRecord) []FundInput {
		return []FundInput{{Label: "fund A", Weight: weight, Records: recs}}
	}

	cases := []struct {
		name    string
		funds   []FundInput
		amount  float64
		wantErr string
	}{
		{
			name:    "no funds",
			funds:   nil,
			amount:  10000,
			wantErr: "no funds provided",
		},
		{
			name:    "zero amount",
			funds:   oneFund(1, models.NAVRecord{Date: time.Now(), NAV: 100}),
			amount:  0,
			wantErr: "amount must be positive",
		},
		{
			name:    "negative amount",
			funds:   oneFund(1, models.NAVRecord{Date: time.Now(), NAV: 100}),
			amount:  -500,
			wantErr: "amount must be positive",
		},
		{
			name:    "negative weight",
			funds:   oneFund(-1, models.NAVRecord{Date: time.Now(), NAV: 100}),
			amount:  10000,
			wantErr: "fund weight cannot be negative",
		},
		{
			name:    "weights sum to zero",
			funds:   oneFund(0, models.NAVRecord{Date: time.Now(), NAV: 100}),
			amount:  10000,
			wantErr: "basket weights sum to zero",
		},
		{
			name:    "fund with no records",
			funds:   oneFund(1),
			amount:  10000,
			wantErr: "no NAV data in the requested range",
		},
		{
			name:    "single data point",
			funds:   oneFund(1, models.NAVRecord{Date: time.Now(), NAV: 100}),
			amount:  10000,
			wantErr: "insufficient NAV data points",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Simulate(tc.funds, tc.amount, false)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("error %v is not a ValidationError", err)
			}
		})
	}
}
