package consumer

import (
	"testing"
)

func TestBuildNotification_OrderConfirmed(t *testing.T) {
	h := &Handler{}
	payload := map[string]string{"order_id": "ord-123", "user_id": "usr-456"}
	n, ok := h.buildNotification("order.confirmed", payload)
	if !ok {
		t.Fatal("expected notification to be built")
	}
	if n.UserID != "usr-456" {
		t.Errorf("expected user_id usr-456, got %s", n.UserID)
	}
	if n.Subject == "" {
		t.Error("expected non-empty subject")
	}
}

func TestBuildNotification_PaymentCaptured(t *testing.T) {
	h := &Handler{}
	payload := map[string]string{"order_id": "ord-123", "user_id": "usr-456", "amount": "5000", "currency": "INR"}
	n, ok := h.buildNotification("payment.captured", payload)
	if !ok {
		t.Fatal("expected notification to be built")
	}
	if n.UserID != "usr-456" {
		t.Errorf("expected user_id usr-456, got %s", n.UserID)
	}
}

func TestBuildNotification_InventoryReserved_Suppressed(t *testing.T) {
	h := &Handler{}
	_, ok := h.buildNotification("inventory.reserved", map[string]string{})
	if ok {
		t.Error("inventory.reserved should produce no notification")
	}
}

func TestBuildNotification_UnknownEvent(t *testing.T) {
	h := &Handler{}
	_, ok := h.buildNotification("unknown.event", map[string]string{})
	if ok {
		t.Error("unknown event should produce no notification")
	}
}

func TestFormatAmount_INR(t *testing.T) {
	got := formatAmount("150050", "INR")
	if got != "₹1,500.50" {
		t.Errorf("expected ₹1,500.50, got %s", got)
	}
}

func TestFormatAmount_JPY(t *testing.T) {
	got := formatAmount("1500", "JPY")
	if got != "¥1,500" {
		t.Errorf("expected ¥1,500, got %s", got)
	}
}

func TestFormatAmount_Invalid(t *testing.T) {
	got := formatAmount("not-a-number", "USD")
	if got != "not-a-number USD" {
		t.Errorf("unexpected fallback: %s", got)
	}
}
