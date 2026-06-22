package consumer

import (
	"testing"
)

func TestFormatAmount_USD(t *testing.T) {
	got := formatAmount("123456", "USD")
	want := "$1,234.56"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_JPY_ZeroDecimal(t *testing.T) {
	got := formatAmount("1500", "JPY")
	want := "¥1,500"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_EUR(t *testing.T) {
	got := formatAmount("9999", "EUR")
	want := "€99.99"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAmount_InvalidAmount(t *testing.T) {
	got := formatAmount("bad", "USD")
	// Should return the raw string rather than crash
	if got == "" {
		t.Error("expected non-empty fallback for bad input")
	}
}
