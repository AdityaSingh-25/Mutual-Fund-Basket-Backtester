package compute

func MaxDrawdown(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	peak := values[0]
	maxDrawdown := 0.0

	for _, value := range values {
		if value > peak {
			peak = value
		}

		drawdown := ((peak - value) / peak) * 100

		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}
