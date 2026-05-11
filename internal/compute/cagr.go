package compute

import "math"

func CAGR(startValue, endValue float64, years float64) float64 {
	if startValue <= 0 || years <= 0 {
		return 0
	}

	return (math.Pow(endValue/startValue, 1/years) - 1) * 100
}
