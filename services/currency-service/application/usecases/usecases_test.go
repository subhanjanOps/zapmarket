package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeCurrencyRepo struct {
	currencies []entities.Currency
	toggleErr  error
}

func (f *fakeCurrencyRepo) ListEnabled(_ context.Context) ([]entities.Currency, error) {
	var out []entities.Currency
	for _, c := range f.currencies {
		if c.Enabled {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCurrencyRepo) Get(_ context.Context, code string) (entities.Currency, error) {
	for _, c := range f.currencies {
		if c.Code == code {
			return c, nil
		}
	}
	return entities.Currency{}, errors.New("not found")
}

func (f *fakeCurrencyRepo) Toggle(_ context.Context, code string, enabled bool) error {
	if f.toggleErr != nil {
		return f.toggleErr
	}
	for i, c := range f.currencies {
		if c.Code == code {
			f.currencies[i].Enabled = enabled
			return nil
		}
	}
	return errors.New("not found")
}

type fakeRatesRepo struct {
	rates []entities.ExchangeRate
	asOf  time.Time
}

func (f *fakeRatesRepo) LatestByBase(_ context.Context, base string) ([]entities.ExchangeRate, time.Time, error) {
	var out []entities.ExchangeRate
	for _, r := range f.rates {
		if r.Base == base {
			out = append(out, r)
		}
	}
	return out, f.asOf, nil
}

func (f *fakeRatesRepo) UpsertLatest(_ context.Context, rates []entities.ExchangeRate, asOf time.Time) error {
	f.rates = rates
	f.asOf = asOf
	return nil
}

type fakeCache struct {
	data map[string]ports.CachedRates
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string]ports.CachedRates{}} }

func (f *fakeCache) Get(_ context.Context, base string) (ports.CachedRates, bool, error) {
	v, ok := f.data[base]
	return v, ok, nil
}

func (f *fakeCache) Set(_ context.Context, base string, r ports.CachedRates, _ time.Duration) error {
	f.data[base] = r
	return nil
}

type fakeProvider struct {
	rateSet ports.RateSet
	err     error
}

func (f *fakeProvider) FetchLatest(_ context.Context, _ string) (ports.RateSet, error) {
	return f.rateSet, f.err
}

// ── ListCurrencies tests ─────────────────────────────────────────────────────

func TestListCurrencies_ReturnsOnlyEnabled(t *testing.T) {
	repo := &fakeCurrencyRepo{currencies: []entities.Currency{
		{Code: "USD", Name: "US Dollar", Enabled: true, Decimals: 2},
		{Code: "JPY", Name: "Japanese Yen", Enabled: false, Decimals: 0},
	}}
	uc := usecases.NewListCurrenciesUseCase(repo)
	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Code != "USD" {
		t.Errorf("expected [USD], got %v", result)
	}
}

// ── GetRates tests ────────────────────────────────────────────────────────────

func TestGetRates_ServesCachedRates(t *testing.T) {
	cache := newFakeCache()
	asOf := time.Now().Add(-30 * time.Minute)
	_ = cache.Set(context.Background(), "USD", ports.CachedRates{
		Base: "USD", AsOf: asOf, Rates: map[string]float64{"EUR": 0.92},
	}, time.Hour)

	uc := usecases.NewGetRatesUseCase(
		&fakeRatesRepo{},
		cache,
		time.Hour,
		24*time.Hour,
	)
	result, err := uc.Execute(context.Background(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	if result.Rates["EUR"] != 0.92 {
		t.Errorf("expected EUR rate 0.92, got %v", result.Rates["EUR"])
	}
	if result.Stale {
		t.Error("expected not stale")
	}
}

func TestGetRates_TooStaleReturns503Error(t *testing.T) {
	asOf := time.Now().Add(-25 * time.Hour)
	repo := &fakeRatesRepo{
		rates: []entities.ExchangeRate{{Base: "USD", Quote: "EUR", Rate: 0.92, AsOf: asOf}},
		asOf:  asOf,
	}
	uc := usecases.NewGetRatesUseCase(repo, newFakeCache(), time.Hour, 24*time.Hour)
	_, err := uc.Execute(context.Background(), "USD")
	var staleErr *usecases.ErrRatesTooStale
	if !errors.As(err, &staleErr) {
		t.Errorf("expected ErrRatesTooStale, got %v", err)
	}
}

func TestGetRates_MarksStaleWhenOlderThan2xInterval(t *testing.T) {
	asOf := time.Now().Add(-3 * time.Hour)
	repo := &fakeRatesRepo{
		rates: []entities.ExchangeRate{{Base: "USD", Quote: "EUR", Rate: 0.92, AsOf: asOf}},
		asOf:  asOf,
	}
	uc := usecases.NewGetRatesUseCase(repo, newFakeCache(), time.Hour, 24*time.Hour)
	result, err := uc.Execute(context.Background(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Stale {
		t.Error("expected stale=true when rates are older than 2x refresh interval")
	}
}

// ── IngestRates tests ─────────────────────────────────────────────────────────

func TestIngestRates_StoresAndCachesRates(t *testing.T) {
	provider := &fakeProvider{rateSet: ports.RateSet{
		Base: "USD", AsOf: time.Now(),
		Rates: map[string]float64{"EUR": 0.92, "JPY": 151.0},
	}}
	repo := &fakeRatesRepo{}
	cache := newFakeCache()
	uc := usecases.NewIngestRatesUseCase(provider, repo, cache, time.Hour, noopLogger())

	if err := uc.Execute(context.Background(), "USD"); err != nil {
		t.Fatal(err)
	}
	if len(repo.rates) != 2 {
		t.Errorf("expected 2 rates persisted, got %d", len(repo.rates))
	}
	if _, ok, _ := cache.Get(context.Background(), "USD"); !ok {
		t.Error("expected cache to be populated after ingest")
	}
}

// ── ToggleCurrency tests ──────────────────────────────────────────────────────

func TestToggleCurrency_DisablesExistingCurrency(t *testing.T) {
	repo := &fakeCurrencyRepo{currencies: []entities.Currency{
		{Code: "USD", Enabled: true},
	}}
	uc := usecases.NewToggleCurrencyUseCase(repo)
	if err := uc.Execute(context.Background(), "USD", false); err != nil {
		t.Fatal(err)
	}
	if repo.currencies[0].Enabled {
		t.Error("expected USD to be disabled")
	}
}

func TestToggleCurrency_ErrorOnUnknownCode(t *testing.T) {
	repo := &fakeCurrencyRepo{}
	uc := usecases.NewToggleCurrencyUseCase(repo)
	if err := uc.Execute(context.Background(), "XYZ", false); err == nil {
		t.Error("expected error for unknown currency code")
	}
}
