package compute

import (
	"math"
	"testing"
)

func TestMaxDrawdown(t *testing.T) {
	cases := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"peak-to-trough", []float64{100, 120, 110, 80, 130}, 33.33},
		{"monotonically increasing", []float64{100, 110, 120, 130}, 0.0},
		{"monotonically decreasing", []float64{100, 80, 60, 40}, 60.0},
		{"single element", []float64{100}, 0.0},
		{"empty slice", []float64{}, 0.0},
		{"recovery after full drop", []float64{100, 50, 100}, 50.0},
		{"flat", []float64{100, 100, 100}, 0.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MaxDrawdown(tc.values)
			if math.IsNaN(got) || math.Abs(got-tc.want) > 0.1 {
				t.Errorf("MaxDrawdown(%v) = %.4f, want %.4f", tc.values, got, tc.want)
			}
		})
	}
}
