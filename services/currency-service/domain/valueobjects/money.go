package valueobjects

// Money holds an amount in minor units (cents) and its currency.
// All amounts stored and settled in the system are USD cents; this type
// is used for conversion math only (display/quote, never settlement).
type Money struct {
	MinorUnits int64
	Currency   CurrencyCode
}

func NewMoney(minorUnits int64, currency CurrencyCode) Money {
	return Money{MinorUnits: minorUnits, Currency: currency}
}
