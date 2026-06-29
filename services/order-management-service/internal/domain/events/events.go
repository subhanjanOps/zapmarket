// Package events holds Kafka event structs consumed and produced by the
// order-management-service saga.  They live in the domain/events layer so both
// the service and consumer packages can import them without circular deps.
package events

import "time"

// CheckoutRequestedEvent is published to checkout.requested when a new order
// is created. Downstream services (inventory, payment) react to this event.
type CheckoutRequestedEvent struct {
	OrderID         string         `json:"order_id"`
	UserID          string         `json:"user_id"`
	Items           []CheckoutItem `json:"items"`
	RequestedAt     time.Time      `json:"requested_at"`
	AmountCents     int64          `json:"amount_cents"`
	Currency        string         `json:"currency"`
	PaymentMethodID string         `json:"payment_method_id"`
}

// CheckoutItem is a single line item within a CheckoutRequestedEvent.
type CheckoutItem struct {
	SkuID    string `json:"sku_id"`
	Quantity int    `json:"quantity"`
}

// InventoryReservedEvent is consumed from inventory.reserved.
// It carries the reservation IDs for each SKU so the order service
// can persist them on order_items for later compensation.
type InventoryReservedEvent struct {
	OrderID      string           `json:"order_id"`
	Reservations []ReservationRef `json:"reservations"`
}

// ReservationRef ties a SKU to its reservation ID.
type ReservationRef struct {
	SKUID         string `json:"sku_id"`
	ReservationID string `json:"reservation_id"`
}

// PaymentCapturedEvent is consumed from payment.captured.
type PaymentCapturedEvent struct {
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id"`
	CapturedAt time.Time `json:"captured_at"`
}

// PaymentFailedEvent is consumed from payment.failed.
type PaymentFailedEvent struct {
	OrderID  string    `json:"order_id"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}

// InventoryReservationFailedEvent is consumed from inventory.reservation_failed.
type InventoryReservationFailedEvent struct {
	OrderID  string    `json:"order_id"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}
