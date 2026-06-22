package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

type oerResponse struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

// OpenExchangeRatesProvider fetches exchange rates from open.er-api.com (free tier, no key).
type OpenExchangeRatesProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewOpenExchangeRatesProvider(baseURL string) *OpenExchangeRatesProvider {
	return &OpenExchangeRatesProvider{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *OpenExchangeRatesProvider) FetchLatest(ctx context.Context, base string) (ports.RateSet, error) {
	url := fmt.Sprintf("%s/v6/latest/%s", p.baseURL, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ports.RateSet{}, fmt.Errorf("oer: build request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ports.RateSet{}, fmt.Errorf("oer: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ports.RateSet{}, fmt.Errorf("oer: upstream status %d", resp.StatusCode)
	}

	var r oerResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return ports.RateSet{}, fmt.Errorf("oer: decode: %w", err)
	}

	if r.Rates == nil {
		r.Rates = make(map[string]float64)
	}
	r.Rates[base] = 1.0

	return ports.RateSet{
		Base:  base,
		AsOf:  time.Now().UTC().Truncate(24 * time.Hour),
		Rates: r.Rates,
	}, nil
}
