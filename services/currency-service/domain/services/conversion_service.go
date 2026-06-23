package services

import (
	"fmt"
	"math"
)

// zeroDecimal lists ISO 4217 currencies with no fractional minor units.
// These are divided by 1 rather than 100 when converting from USD cents.
var zeroDecimal = map[string]bool{
	"BIF": true,
	"CLP": true,
	"GNF": true,
	"HUF": true,
	"IDR": true,
	"ISK": true,
	"JPY": true,
	"KMF": true,
	"KRW": true,
	"MGA": true,
	"PYG": true,
	"RWF": true,
	"TWD": true,
	"UGX": true,
	"VND": true,
	"VUV": true,
	"XAF": true,
	"XOF": true,
	"XPF": true,
}

// ConversionService converts USD-cent amounts to a display amount in a target currency.
// Inputs are minor units (USD cents). Returns a float64 display amount.
// This service is pure (no I/O) and is available for a future /convert endpoint.
type ConversionService struct{}

func NewConversionService() *ConversionService { return &ConversionService{} }

// Convert converts usdCents to the target currency using the provided rate map
// (keyed by ISO 4217 code, values are multipliers relative to USD).
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
