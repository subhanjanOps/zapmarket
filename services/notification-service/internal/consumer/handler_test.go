package consumer_test

import (
	"testing"

	"github.com/zapmarket/zapmarket/services/notification-service/internal/consumer"
)

func TestFormatAmount_USD(t *testing.T) {
	got := consumer.FormatAmountForTest("123456", "USD")
	want := "$1,234.56"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_JPY_ZeroDecimal(t *testing.T) {
	got := consumer.FormatAmountForTest("150000", "JPY")
	want := "¥1,500"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_EUR(t *testing.T) {
	got := consumer.FormatAmountForTest("9999", "EUR")
	want := "€99.99"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_InvalidAmount(t *testing.T) {
	got := consumer.FormatAmountForTest("bad", "USD")
	// Should return the raw string rather than crash
	if got == "" {
		t.Error("expected non-empty fallback for bad input")
	}
}
