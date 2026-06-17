package domain

import (
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

type OrderStatus string

const (
	OrderPending   OrderStatus = "PENDING"
	OrderReserved  OrderStatus = "RESERVED"
	OrderPaid      OrderStatus = "PAID"
	OrderConfirmed OrderStatus = "CONFIRMED"
	OrderCancelled OrderStatus = "CANCELLED"
)

// allowedTransitions is the authoritative FSM definition. Any transition not
// listed here is illegal and will be rejected by Transition.
var allowedTransitions = map[OrderStatus]map[OrderStatus]struct{}{
	OrderPending:  {OrderReserved: {}, OrderCancelled: {}},
	OrderReserved: {OrderPaid: {}, OrderCancelled: {}},
	OrderPaid:     {OrderConfirmed: {}},
}

type Order struct {
	ID             uuid.UUID   `json:"id"`
	UserID         uuid.UUID   `json:"user_id"`
	IdempotencyKey uuid.UUID   `json:"idempotency_key"`
	Status         OrderStatus `json:"status"`
	TotalAmount    int64       `json:"total_amount"`
	Currency       string      `json:"currency"`
	PaymentID      *uuid.UUID  `json:"payment_id,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
}

// Transition returns a new OrderStatus if the transition from o.Status → next
// is allowed, or a Validation error if it is not.
func (o *Order) Transition(next OrderStatus) error {
	allowed, ok := allowedTransitions[o.Status]
	if !ok {
		return pkgerrors.NewValidation("INVALID_TRANSITION", "order is in a terminal state: "+string(o.Status))
	}
	if _, ok := allowed[next]; !ok {
		return pkgerrors.NewValidation("INVALID_TRANSITION",
			"cannot transition order from "+string(o.Status)+" to "+string(next))
	}
	return nil
}

type OrderItem struct {
	ID            uuid.UUID  `json:"id"`
	OrderID       uuid.UUID  `json:"order_id"`
	SKUID         uuid.UUID  `json:"sku_id"`
	Quantity      int        `json:"quantity"`
	UnitPrice     int64      `json:"unit_price"`
	ReservationID *uuid.UUID `json:"reservation_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
