package errors

import "errors"

var (
	ErrInsufficientStock      = errors.New("insufficient stock to fulfill reservation")
	ErrReservationNotFound    = errors.New("reservation not found")
	ErrReservationNotReserved = errors.New("reservation is not in RESERVED state")
	ErrInventoryNotFound      = errors.New("no inventory record for this SKU")
)
