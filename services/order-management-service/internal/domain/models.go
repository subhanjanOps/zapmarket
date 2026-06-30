package domain

import (
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

type OrderStatus string

const (
	OrderPending       OrderStatus = "PENDING"
	OrderReserved      OrderStatus = "RESERVED"
	OrderPaid          OrderStatus = "PAID"
	OrderConfirmed     OrderStatus = "CONFIRMED"
	OrderCancelled     OrderStatus = "CANCELLED"
	OrderDeliveryFailed OrderStatus = "DELIVERY_FAILED"
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
	SagaStatus     string      `json:"saga_status,omitempty"`
	TotalAmount    int64       `json:"total_amount"`
	DiscountPaise  int64       `json:"discount_paise,omitempty"`
	Currency       string      `json:"currency"`
	CouponCode     *string     `json:"coupon_code,omitempty"`
	PaymentID      *uuid.UUID  `json:"payment_id,omitempty"`
	// Delivery address
	DeliveryFullName    *string `json:"delivery_full_name,omitempty"`
	DeliveryPhone       *string `json:"delivery_phone,omitempty"`
	DeliveryAddressLine1 *string `json:"delivery_address_line1,omitempty"`
	DeliveryCity        *string `json:"delivery_city,omitempty"`
	DeliveryPincode     *string `json:"delivery_pincode,omitempty"`
	DeliveryCountry     string  `json:"delivery_country,omitempty"`
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
	SellerID      *uuid.UUID `json:"seller_id,omitempty"`
	Quantity      int        `json:"quantity"`
	UnitPrice     int64      `json:"unit_price"`
	ReservationID *uuid.UUID `json:"reservation_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
