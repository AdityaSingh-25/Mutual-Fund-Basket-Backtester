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

	result, err := Run(basketID, "2021-01-01", "2022-01-01", 10000)
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
}

func TestRunErrors(t *testing.T) {
	testsupport.RequireDB(t)

	t.Run("nonexistent basket", func(t *testing.T) {
		if _, err := Run(99_999_999, "2021-01-01", "2022-01-01", 10000); err == nil {
			t.Error("expected an error for a nonexistent basket")
		}
	})

	t.Run("invalid start date", func(t *testing.T) {
		if _, err := Run(1, "not-a-date", "2022-01-01", 10000); err == nil {
			t.Error("expected an error for an invalid start date")
		}
	})

	t.Run("end before start", func(t *testing.T) {
		if _, err := Run(1, "2022-01-01", "2021-01-01", 10000); err == nil {
			t.Error("expected an error when end_date precedes start_date")
		}
	})
}
