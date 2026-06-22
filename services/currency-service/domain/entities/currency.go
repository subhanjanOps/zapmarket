package entities

import "time"

// Currency is the supported-currency catalogue entry.
type Currency struct {
	Code      string
	Name      string
	Flag      string
	Decimals  int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExchangeRate is one row of the exchange_rates table.
type ExchangeRate struct {
	Base      string
	Quote     string
	Rate      float64
	AsOf      time.Time
	FetchedAt time.Time
}
