package valueobjects

import "fmt"

// Rate is an exchange rate as a decimal multiplier (e.g. 1 USD = 82.5 INR → rate 82.5).
type Rate float64

func NewRate(v float64) (Rate, error) {
	if v <= 0 {
		return 0, fmt.Errorf("rate must be positive, got %f", v)
	}
	return Rate(v), nil
}

func (r Rate) Float64() float64 { return float64(r) }
