package dto

// RatesHistoryDTO is the response for a historical rates lookup.
type RatesHistoryDTO struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}
