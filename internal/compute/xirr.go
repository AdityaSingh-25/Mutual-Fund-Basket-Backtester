package compute

import (
	"math"
	"time"
)

type Cashflow struct {
	Date   time.Time
	Amount float64
}

func XIRR(cashflows []Cashflow) float64 {
	if len(cashflows) == 0 {
		return 0
	}

	guess := 0.1

	for i := 0; i < 100; i++ {
		fValue := 0.0
		derivative := 0.0

		for _, cf := range cashflows {
			years := cf.Date.Sub(cashflows[0].Date).Hours() / 24 / 365

			denominator := math.Pow(1+guess, years)

			fValue += cf.Amount / denominator

			derivative += (-years * cf.Amount) /
				(math.Pow(1+guess, years+1))
		}

		newGuess := guess - (fValue / derivative)

		if math.Abs(newGuess-guess) < 0.000001 {
			return newGuess * 100
		}

		guess = newGuess
	}

	return guess * 100
}
