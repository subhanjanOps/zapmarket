package errors

import "errors"

var (
	ErrReviewNotFound      = errors.New("review not found")
	ErrReturnNotFound      = errors.New("return request not found")
	ErrAlreadyReviewed     = errors.New("user has already reviewed this SKU for this order")
	ErrReturnNotEligible   = errors.New("order item is not eligible for return")
)
