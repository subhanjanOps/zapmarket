package services

import (
	"fmt"
	"math"
)

// zeroDecimal currencies have no fractional minor units (1 JPY = 1 JPY, not 100).
var zeroDecimal = map[string]bool{
	"JPY": true,
	"KRW": true,
	"IDR": true,
}

// ConversionService converts USD-cent amounts to a display amount in a target currency.
// All inputs are minor units (USD cents). Returns a float64 display amount.
// This service is pure (no I/O) and fully unit-testable.
type ConversionService struct{}

func NewConversionService() *ConversionService { return &ConversionService{} }

// Convert converts usdCents to the target currency using the provided rate map
// (keyed by ISO 4217 code, values are multipliers relative to USD).
// Returns the display amount (already in the target currency's major/minor unit).
func (s *ConversionService) Convert(usdCents int64, toCurrency string, rates map[string]float64) (float64, error) {
	rate, ok := rates[toCurrency]
	if !ok {
		return 0, fmt.Errorf("no rate available for %s", toCurrency)
	}
	if rate <= 0 {
		return 0, fmt.Errorf("invalid rate %f for %s", rate, toCurrency)
	}

	usdMajor := float64(usdCents) / 100.0
	converted := usdMajor * rate

	if zeroDecimal[toCurrency] {
		return math.Round(converted), nil
	}
	return math.Round(converted*100) / 100, nil
}

// IsZeroDecimal reports whether the currency has no fractional minor units.
func (s *ConversionService) IsZeroDecimal(code string) bool {
	return zeroDecimal[code]
}
