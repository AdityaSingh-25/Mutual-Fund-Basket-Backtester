package compute

import "testing"

func TestMaxDrawdown(t *testing.T) {
	values := []float64{100, 120, 110, 80, 130}

	result := MaxDrawdown(values)

	expected := 33.33

	if result < expected-0.1 || result > expected+0.1 {
		t.Errorf("Expected %.2f, got %.2f", expected, result)
	}
}
