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
	ID             uuid.UUID
	UserID         uuid.UUID
	IdempotencyKey uuid.UUID
	Status         OrderStatus
	TotalAmount    int64
	Currency       string
	PaymentID      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
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
	ID            uuid.UUID
	OrderID       uuid.UUID
	SKUID         uuid.UUID
	Quantity      int
	UnitPrice     int64
	ReservationID *uuid.UUID
	CreatedAt     time.Time
}
