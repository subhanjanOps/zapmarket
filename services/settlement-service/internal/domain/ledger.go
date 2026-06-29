package domain

import "time"

type EntryType string

const (
	EntryTypeCredit          EntryType = "CREDIT_SALE"
	EntryTypeDebitRefund     EntryType = "DEBIT_REFUND"
	EntryTypeDebitCommission EntryType = "DEBIT_COMMISSION"
	EntryTypePayout          EntryType = "DEBIT_PAYOUT"
)

type LedgerEntry struct {
	ID              string
	SellerID        string
	OrderID         string
	PaymentID       string
	EntryType       EntryType
	AmountPaise     int64
	CommissionPaise int64
	NetPaise        int64
	Currency        string
	Note            string
	CreatedAt       time.Time
}

type SellerBalance struct {
	SellerID      string
	PendingPaise  int64
	PaidOutPaise  int64
	Currency      string
	LastUpdatedAt time.Time
}

type Payout struct {
	ID               string
	SellerID         string
	AmountPaise      int64
	Currency         string
	RazorpayPayoutID string
	Status           string
	InitiatedAt      time.Time
	ProcessedAt      *time.Time
}
