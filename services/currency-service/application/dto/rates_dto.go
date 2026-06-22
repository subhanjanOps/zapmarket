package dto

import "time"

// RatesDTO is the response payload for GET /api/v1/currencies/rates.
// Date is an alias for AsOf kept for backward compatibility with existing consumers.
type RatesDTO struct {
	Base  string             `json:"base"`
	AsOf  time.Time          `json:"as_of"`
	Date  string             `json:"date"`  // ISO date alias for as_of (backward compat)
	Stale bool               `json:"stale"`
	Rates map[string]float64 `json:"rates"`
}
