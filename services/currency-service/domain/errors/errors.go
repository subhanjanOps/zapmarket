package errors

import "fmt"

// ErrCurrencyNotFound is returned when a currency code does not exist in the catalogue.
type ErrCurrencyNotFound struct{ Code string }

func (e *ErrCurrencyNotFound) Error() string {
	return fmt.Sprintf("currency %q not found", e.Code)
}

// ErrNoRatesHistory is returned when no historical rates exist for the requested base and date.
type ErrNoRatesHistory struct {
	Base string
	Date string
}

func (e *ErrNoRatesHistory) Error() string {
	return fmt.Sprintf("no rate history for %s on %s", e.Base, e.Date)
}
