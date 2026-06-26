package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "PENDING"
	PaymentAuthorised        PaymentStatus = "AUTHORISED"
	PaymentCaptured          PaymentStatus = "CAPTURED"
	PaymentFailed            PaymentStatus = "FAILED"
	PaymentRefunded          PaymentStatus = "REFUNDED"
	PaymentPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
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
	LedgerDebit  LedgerEntryType = "DEBIT"
	LedgerCredit LedgerEntryType = "CREDIT"
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
	RefundPending   RefundStatus = "PENDING"
	RefundProcessed RefundStatus = "PROCESSED"
	RefundFailed    RefundStatus = "FAILED"
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
