package dto

// CurrencyDTO is the public representation of a currency (enabled-only; no Enabled field exposed).
type CurrencyDTO struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Flag     string `json:"flag"`
	Decimals int    `json:"decimals"`
}
