package external_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/infrastructure/external"
)

type stubProvider struct {
	result ports.RateSet
	err    error
}

func (s *stubProvider) FetchLatest(_ context.Context, base string) (ports.RateSet, error) {
	return s.result, s.err
}

func TestFallbackProvider_UsesSecondaryOnPrimaryError(t *testing.T) {
	primary := &stubProvider{err: errors.New("primary down")}
	secondary := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.9}}}

	fp := external.NewFallbackProvider(primary, secondary)
	got, err := fp.FetchLatest(context.Background(), "USD")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Rates["EUR"] != 0.9 {
		t.Errorf("expected EUR=0.9, got %v", got.Rates["EUR"])
	}
}

func TestFallbackProvider_UsesPrimaryWhenHealthy(t *testing.T) {
	primary := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.85}}}
	secondary := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.9}}}

	fp := external.NewFallbackProvider(primary, secondary)
	got, err := fp.FetchLatest(context.Background(), "USD")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Rates["EUR"] != 0.85 {
		t.Errorf("expected EUR=0.85 from primary, got %v", got.Rates["EUR"])
	}
}

func TestFallbackProvider_ErrorsWhenBothFail(t *testing.T) {
	primary := &stubProvider{err: errors.New("primary down")}
	secondary := &stubProvider{err: errors.New("secondary down")}

	fp := external.NewFallbackProvider(primary, secondary)
	_, err := fp.FetchLatest(context.Background(), "USD")
	if err == nil {
		t.Fatal("expected error when both providers fail")
	}
}
