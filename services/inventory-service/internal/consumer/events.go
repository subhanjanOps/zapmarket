package consumer

import "time"

// CheckoutRequestedEvent is published by the order-management-service when a
// buyer initiates checkout. The inventory-service consumes this event and
// attempts to reserve stock for every item.
type CheckoutRequestedEvent struct {
	OrderID         string         `json:"order_id"`
	UserID          string         `json:"user_id"`
	Items           []CheckoutItem `json:"items"`
	RequestedAt     time.Time      `json:"requested_at"`
	AmountCents     int64          `json:"amount_cents"`
	Currency        string         `json:"currency"`
	PaymentMethodID string         `json:"payment_method_id"`
}

// CheckoutItem is a single line item inside a CheckoutRequestedEvent.
type CheckoutItem struct {
	SKUID    string `json:"sku_id"`
	Quantity int    `json:"quantity"`
}

// InventoryReservedEvent is published when ALL items in an order have been
// successfully reserved. The payment-service listens for this event.
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

// InventoryReservationFailedEvent is published when stock cannot be reserved
// for one or more items. Any previously-reserved items are compensated before
// this event is sent.
type InventoryReservationFailedEvent struct {
	OrderID  string    `json:"order_id"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}
