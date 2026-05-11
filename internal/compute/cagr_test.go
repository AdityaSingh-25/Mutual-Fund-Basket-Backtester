package compute

import "testing"

func TestCAGR(t *testing.T) {
	result := CAGR(1000, 2000, 5)

	expected := 14.87

	if result < expected-0.1 || result > expected+0.1 {
		t.Errorf("Expected %.2f, got %.2f", expected, result)
	}
}
