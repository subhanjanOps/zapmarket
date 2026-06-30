package domain

import "time"

type DiscountType string

const (
	DiscountTypePercent    DiscountType = "PERCENT"
	DiscountTypeFixedPaise DiscountType = "FIXED_PAISE"
)

type Coupon struct {
	ID             string
	Code           string // always stored UPPERCASE
	DiscountType   DiscountType
	DiscountValue  int64      // percent (1-100) or fixed paise amount
	MinOrderPaise  int64      // minimum cart value to apply
	MaxUsesTotal   int        // 0 = unlimited
	MaxUsesPerUser int        // 0 = unlimited
	ExpiresAt      *time.Time
	StartsAt       *time.Time
	EndsAt         *time.Time
	IsActive       bool
	CreatedAt      time.Time
}

func (c *Coupon) Apply(cartTotalPaise int64) (discountPaise int64) {
	if cartTotalPaise < c.MinOrderPaise {
		return 0
	}
	switch c.DiscountType {
	case DiscountTypePercent:
		return (cartTotalPaise * c.DiscountValue) / 100
	case DiscountTypeFixedPaise:
		if c.DiscountValue > cartTotalPaise {
			return cartTotalPaise
		}
		return c.DiscountValue
	}
	return 0
}
