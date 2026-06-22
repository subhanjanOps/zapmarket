package services_test

import (
	"testing"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/services"
)

func TestConversionService_Convert(t *testing.T) {
	svc := services.NewConversionService()
	rates := map[string]float64{
		"USD": 1.0,
		"EUR": 0.92,
		"INR": 83.5,
		"JPY": 151.0,
		"KRW": 1350.0,
		"IDR": 15800.0,
	}

	tests := []struct {
		name     string
		cents    int64
		currency string
		want     float64
		wantErr  bool
	}{
		{"identity USD", 1000, "USD", 10.0, false},
		{"EUR conversion", 1000, "EUR", 9.2, false},
		{"INR conversion", 1000, "INR", 835.0, false},
		{"JPY zero-decimal", 1000, "JPY", 1510, false},
		{"KRW zero-decimal", 1000, "KRW", 13500, false},
		{"IDR zero-decimal", 1000, "IDR", 158000, false},
		{"missing rate", 1000, "XYZ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Convert(tt.cents, tt.currency, rates)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Convert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Convert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConversionService_IsZeroDecimal(t *testing.T) {
	svc := services.NewConversionService()
	if !svc.IsZeroDecimal("JPY") {
		t.Error("JPY should be zero-decimal")
	}
	if !svc.IsZeroDecimal("KRW") {
		t.Error("KRW should be zero-decimal")
	}
	if !svc.IsZeroDecimal("IDR") {
		t.Error("IDR should be zero-decimal")
	}
	if svc.IsZeroDecimal("USD") {
		t.Error("USD should not be zero-decimal")
	}
	if svc.IsZeroDecimal("EUR") {
		t.Error("EUR should not be zero-decimal")
	}
}
