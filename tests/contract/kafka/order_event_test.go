package kafka_test

import (
	"encoding/json"
	"testing"
	"time"
)

// OrderConfirmedEvent is the canonical shape produced by order-management-service.
type OrderConfirmedEvent struct {
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	PaymentID   string    `json:"payment_id"`
	TotalAmount int64     `json:"total_amount"`
	Currency    string    `json:"currency"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

func TestOrderConfirmedEvent_HasRequiredFields(t *testing.T) {
	raw := `{
		"order_id":     "ord-123",
		"user_id":      "usr-456",
		"payment_id":   "pay-789",
		"total_amount": 99900,
		"currency":     "INR",
		"confirmed_at": "2026-06-29T10:00:00Z"
	}`

	var evt OrderConfirmedEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("event does not match expected schema: %v", err)
	}
	if evt.OrderID == "" {
		t.Error("order_id is required")
	}
	if evt.UserID == "" {
		t.Error("user_id is required")
	}
	if evt.Currency == "" {
		t.Error("currency is required")
	}
	if evt.TotalAmount <= 0 {
		t.Error("total_amount must be positive")
	}
}

// InventoryReservedEvent is the canonical shape produced by inventory-service.
type InventoryReservedEvent struct {
	ReservationID string `json:"reservation_id"`
	OrderID       string `json:"order_id"`
	SKUID         string `json:"sku_id"`
	Qty           int    `json:"qty"`
	Status        string `json:"status"`
}

func TestInventoryReservedEvent_HasRequiredFields(t *testing.T) {
	raw := `{
		"reservation_id": "res-001",
		"order_id":       "ord-123",
		"sku_id":         "sku-abc",
		"qty":            2,
		"status":         "RESERVED"
	}`

	var evt InventoryReservedEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("event does not match expected schema: %v", err)
	}
	if evt.ReservationID == "" {
		t.Error("reservation_id is required")
	}
	if evt.Status != "RESERVED" {
		t.Errorf("expected status RESERVED, got %s", evt.Status)
	}
}

// PaymentCapturedEvent is the canonical shape produced by payment-service.
type PaymentCapturedEvent struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
}

func TestPaymentCapturedEvent_HasRequiredFields(t *testing.T) {
	raw := `{
		"payment_id": "pay-789",
		"order_id":   "ord-123",
		"amount":     99900,
		"currency":   "INR",
		"status":     "PAYMENT_SUCCESSFUL"
	}`

	var evt PaymentCapturedEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("event does not match expected schema: %v", err)
	}
	if evt.Status != "PAYMENT_SUCCESSFUL" {
		t.Errorf("expected PAYMENT_SUCCESSFUL, got %s", evt.Status)
	}
}
