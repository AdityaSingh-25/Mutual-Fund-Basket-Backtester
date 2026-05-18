package backtest

import (
	"testing"

	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/testsupport"
)

// TestRunIntegration exercises the database-backed Run wrapper end to end,
// mirroring the "two funds, unnormalised weights" pure Simulate case.
func TestRunIntegration(t *testing.T) {
	testsupport.RequireDB(t)

	f1 := testsupport.InsertFund(t, "Run Integration Fund A")
	f2 := testsupport.InsertFund(t, "Run Integration Fund B")

	testsupport.InsertNAV(t, f1, "2021-01-01", 100)
	testsupport.InsertNAV(t, f1, "2022-01-01", 120)
	testsupport.InsertNAV(t, f2, "2021-01-01", 50)
	testsupport.InsertNAV(t, f2, "2022-01-01", 75)

	basketID := testsupport.InsertBasket(t, "Run Integration Basket")
	if err := db.InsertBasketItem(basketID, f1, 60); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}
	if err := db.InsertBasketItem(basketID, f2, 40); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	result, err := Run(basketID, "2021-01-01", "2022-01-01", 10000, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 60 units of A at 120 + 80 units of B at 75 = 7200 + 6000 = 13200.
	if result.FinalValue != 13200 {
		t.Errorf("FinalValue = %v, want 13200", result.FinalValue)
	}
	if result.CAGR != 32 {
		t.Errorf("CAGR = %v, want 32", result.CAGR)
	}
	if result.Mode != "lumpsum" {
		t.Errorf("Mode = %q, want lumpsum", result.Mode)
	}
	if len(result.Series) != 2 {
		t.Errorf("len(Series) = %d, want 2", len(result.Series))
	}
}

// TestRunSIPIntegration verifies a monthly SIP through the DB-backed wrapper:
// four contributions of 1000 at a flat NAV return exactly the money invested.
func TestRunSIPIntegration(t *testing.T) {
	testsupport.RequireDB(t)

	fund := testsupport.InsertFund(t, "Run SIP Fund")
	for _, date := range []string{"2021-01-01", "2021-02-01", "2021-03-01", "2021-04-01"} {
		testsupport.InsertNAV(t, fund, date, 100)
	}

	basketID := testsupport.InsertBasket(t, "Run SIP Basket")
	if err := db.InsertBasketItem(basketID, fund, 100); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	result, err := Run(basketID, "2021-01-01", "2021-04-01", 1000, true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Mode != "sip" {
		t.Errorf("Mode = %q, want sip", result.Mode)
	}
	if result.TotalInvested != 4000 {
		t.Errorf("TotalInvested = %v, want 4000", result.TotalInvested)
	}
	if result.FinalValue != 4000 {
		t.Errorf("FinalValue = %v, want 4000", result.FinalValue)
	}
}

// TestRunFundIntegration checks the single-fund backtest used for benchmarks.
func TestRunFundIntegration(t *testing.T) {
	testsupport.RequireDB(t)

	fund := testsupport.InsertFund(t, "RunFund Benchmark Fund")
	testsupport.InsertNAV(t, fund, "2021-01-01", 100)
	testsupport.InsertNAV(t, fund, "2022-01-01", 150)

	result, err := RunFund(fund, "2021-01-01", "2022-01-01", 10000, false)
	if err != nil {
		t.Fatalf("RunFund: %v", err)
	}

	// 100 units bought at NAV 100, valued at 150 → 15000; 50% over one year.
	if result.FinalValue != 15000 {
		t.Errorf("FinalValue = %v, want 15000", result.FinalValue)
	}
	if result.CAGR != 50 {
		t.Errorf("CAGR = %v, want 50", result.CAGR)
	}
}

func TestRunErrors(t *testing.T) {
	testsupport.RequireDB(t)

	t.Run("nonexistent basket", func(t *testing.T) {
		if _, err := Run(99_999_999, "2021-01-01", "2022-01-01", 10000, false); err == nil {
			t.Error("expected an error for a nonexistent basket")
		}
	})

	t.Run("invalid start date", func(t *testing.T) {
		if _, err := Run(1, "not-a-date", "2022-01-01", 10000, false); err == nil {
			t.Error("expected an error for an invalid start date")
		}
	})

	t.Run("end before start", func(t *testing.T) {
		if _, err := Run(1, "2022-01-01", "2021-01-01", 10000, false); err == nil {
			t.Error("expected an error when end_date precedes start_date")
		}
	})
}
