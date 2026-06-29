package domain

import "time"

type Cart struct {
	UserID    string
	Items     []CartItem
	UpdatedAt time.Time
}

type CartItem struct {
	SKUID        string            `json:"sku_id"`
	ProductID    string            `json:"product_id"`
	ProductName  string            `json:"product_name"`
	VariantAttrs map[string]string `json:"variant_attrs"`
	Quantity     int               `json:"quantity"`
	PriceAtAdd   int64             `json:"price_at_add"`
	Currency     string            `json:"currency"`
	ImageURL     string            `json:"image_url"`
	AddedAt      time.Time         `json:"added_at"`
}

func (i *CartItem) PriceStale(currentPriceCents int64) bool {
	return i.PriceAtAdd != currentPriceCents
}
