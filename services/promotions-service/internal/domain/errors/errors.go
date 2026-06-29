package errors

import "errors"

var (
	ErrCouponNotFound = errors.New("coupon not found")
	ErrCouponInactive = errors.New("coupon is not active")
	ErrCouponExpired  = errors.New("coupon has expired")
)
