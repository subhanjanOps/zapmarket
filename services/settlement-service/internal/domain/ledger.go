package domain

import "time"

type EntryType string

const (
	EntryTypeCredit          EntryType = "CREDIT_SALE"
	EntryTypeDebitRefund     EntryType = "DEBIT_REFUND"
	EntryTypeDebitCommission EntryType = "DEBIT_COMMISSION"
	EntryTypePayout          EntryType = "DEBIT_PAYOUT"
	EntryTypeDebitTDS        EntryType = "DEBIT_TDS"
	EntryTypeDebitGST        EntryType = "DEBIT_GST_COMMISSION"
	EntryTypeDebitReturn     EntryType = "DEBIT_RETURN"
)

type LedgerEntry struct {
	ID                   string
	SellerID             string
	OrderID              string
	PaymentID            string
	EntryType            EntryType
	AmountPaise          int64
	CommissionPaise      int64
	TDSPaise             int64
	GSTOnCommissionPaise int64
	NetPaise             int64
	Currency             string
	Note                 string
	CreatedAt            time.Time
}

type SellerBankAccount struct {
	ID                 string    `json:"id"`
	SellerID           string    `json:"seller_id"`
	AccountHolderName  string    `json:"account_holder_name"`
	AccountNumber      string    `json:"account_number"`
	IFSCCode           string    `json:"ifsc_code"`
	BankName           string    `json:"bank_name"`
	UPIID              string    `json:"upi_id,omitempty"`
	IsVerified         bool      `json:"is_verified"`
	IsPrimary          bool      `json:"is_primary"`
	CreatedAt          time.Time `json:"created_at"`
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
