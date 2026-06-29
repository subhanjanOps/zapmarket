package errors

import "errors"

var (
	ErrCartNotFound = errors.New("cart not found")
	ErrItemNotFound = errors.New("item not found in cart")
	ErrOutOfStock   = errors.New("sku is out of stock")
	ErrPriceChanged = errors.New("price has changed since item was added to cart")
)
