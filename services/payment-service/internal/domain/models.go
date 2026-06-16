package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "pending"
	PaymentAuthorised        PaymentStatus = "authorised"
	PaymentCaptured          PaymentStatus = "captured"
	PaymentFailed            PaymentStatus = "failed"
	PaymentRefunded          PaymentStatus = "refunded"
	PaymentPartiallyRefunded PaymentStatus = "partially_refunded"
)

type Payment struct {
	ID              uuid.UUID
	OrderID         uuid.UUID
	UserID          uuid.UUID
	IdempotencyKey  uuid.UUID
	Status          PaymentStatus
	Amount          int64
	Currency        string
	Gateway         string
	GatewayTxnID    *string
	GatewayResponse json.RawMessage
	FailureReason   *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

type LedgerEntryType string

const (
	LedgerDebit  LedgerEntryType = "debit"
	LedgerCredit LedgerEntryType = "credit"
)

// LedgerEntry is an immutable double-entry ledger row — no UpdatedAt or
// DeletedAt, append-only by design.
type LedgerEntry struct {
	ID          uuid.UUID
	PaymentID   uuid.UUID
	EntryType   LedgerEntryType
	Account     string
	Amount      int64
	Currency    string
	Description string
	CreatedAt   time.Time
}

type RefundStatus string

const (
	RefundPending   RefundStatus = "pending"
	RefundProcessed RefundStatus = "processed"
	RefundFailed    RefundStatus = "failed"
)

type Refund struct {
	ID              uuid.UUID
	PaymentID       uuid.UUID
	OrderID         uuid.UUID
	Amount          int64
	Currency        string
	Reason          string
	Status          RefundStatus
	GatewayRefundID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
