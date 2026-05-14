package compute

import (
	"math"
	"testing"
)

func TestCAGR(t *testing.T) {
	cases := []struct {
		name       string
		start, end float64
		years      float64
		want       float64
	}{
		{"double in 5 years", 1000, 2000, 5, 14.87},
		{"no growth", 1000, 1000, 5, 0.0},
		{"loss", 1000, 500, 4, -15.91},
		{"one year doubling", 1000, 2000, 1, 100.0},
		{"zero start returns 0", 0, 2000, 5, 0},
		{"zero years returns 0", 1000, 2000, 0, 0},
		{"negative years returns 0", 1000, 2000, -1, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CAGR(tc.start, tc.end, tc.years)
			if math.IsNaN(got) || math.Abs(got-tc.want) > 0.1 {
				t.Errorf("CAGR(%.0f, %.0f, %.0f) = %.4f, want %.4f",
					tc.start, tc.end, tc.years, got, tc.want)
			}
		})
	}
}
