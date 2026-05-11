package compute

import (
	"testing"
	"time"
)

func TestXIRR(t *testing.T) {
	cashflows := []Cashflow{
		{
			Date:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount: -1000,
		},
		{
			Date:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount: 1200,
		},
	}

	result := XIRR(cashflows)

	expected := 20.0

	if result < expected-0.5 || result > expected+0.5 {
		t.Errorf("Expected %.2f, got %.2f", expected, result)
	}
}
