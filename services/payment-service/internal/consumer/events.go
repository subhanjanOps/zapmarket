package consumer

import "time"

// InventoryReservedEvent is consumed from the inventory.reserved topic.
// It carries all fields needed to charge the card without a DB lookup.
type InventoryReservedEvent struct {
	OrderID         string           `json:"order_id"`
	UserID          string           `json:"user_id"`
	Reservations    []ReservationRef `json:"reservations"`
	ReservedAt      time.Time        `json:"reserved_at"`
	AmountCents     int64            `json:"amount_cents"`
	Currency        string           `json:"currency"`
	PaymentMethodID string           `json:"payment_method_id"`
}

// ReservationRef ties a SKU to the reservation that was created for it.
type ReservationRef struct {
	SKUID         string `json:"sku_id"`
	ReservationID string `json:"reservation_id"`
	Quantity      int    `json:"quantity"`
}

// PaymentCapturedEvent is published to payment.captured when a charge succeeds.
type PaymentCapturedEvent struct {
	OrderID     string    `json:"order_id"`
	PaymentID   string    `json:"payment_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	CapturedAt  time.Time `json:"captured_at"`
}

// PaymentFailedEvent is published to payment.failed when a charge fails.
type PaymentFailedEvent struct {
	OrderID  string    `json:"order_id"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}
