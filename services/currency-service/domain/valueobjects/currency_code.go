package valueobjects

import (
	"fmt"
	"strings"
)

// CurrencyCode is a validated ISO 4217 3-letter currency code.
type CurrencyCode string

func NewCurrencyCode(s string) (CurrencyCode, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 3 {
		return "", fmt.Errorf("currency code must be 3 characters, got %q", s)
	}
	return CurrencyCode(s), nil
}

func (c CurrencyCode) String() string { return string(c) }
