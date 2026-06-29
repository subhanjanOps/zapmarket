package errors

import "errors"

var (
	ErrInsufficientBalance = errors.New("seller balance below minimum payout threshold")
	ErrPayoutFailed        = errors.New("payout gateway returned failure")
	ErrSellerNotFound      = errors.New("seller account not found")
)
