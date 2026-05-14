package compute

import (
	"math"
	"testing"
	"time"
)

func TestXIRR(t *testing.T) {
	d := func(y, m, day int) time.Time {
		return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	}

	cases := []struct {
		name      string
		cashflows []Cashflow
		want      float64
		tol       float64
	}{
		{
			"20% annual return",
			[]Cashflow{{d(2020, 1, 1), -1000}, {d(2021, 1, 1), 1200}},
			20.0, 0.5,
		},
		{
			"0% return break-even",
			[]Cashflow{{d(2020, 1, 1), -1000}, {d(2021, 1, 1), 1000}},
			0.0, 0.5,
		},
		{
			"multi-year SIP-like",
			[]Cashflow{
				{d(2020, 1, 1), -1000},
				{d(2021, 1, 1), -1000},
				{d(2022, 1, 1), 2300},
			},
			9.68, 0.5,
		},
		{
			"empty returns 0",
			[]Cashflow{},
			0.0, 0.001,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := XIRR(tc.cashflows)
			if math.IsNaN(got) || math.Abs(got-tc.want) > tc.tol {
				t.Errorf("XIRR(...) = %.4f, want %.4f (±%.3f)", got, tc.want, tc.tol)
			}
		})
	}
}
