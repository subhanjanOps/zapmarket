package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

type frankfurterResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// FrankfurterProvider fetches exchange rates from api.frankfurter.app.
type FrankfurterProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewFrankfurterProvider(baseURL string) *FrankfurterProvider {
	return &FrankfurterProvider{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *FrankfurterProvider) FetchLatest(ctx context.Context, base string) (ports.RateSet, error) {
	url := fmt.Sprintf("%s/latest?from=%s", p.baseURL, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ports.RateSet{}, fmt.Errorf("frankfurter: build request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ports.RateSet{}, fmt.Errorf("frankfurter: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ports.RateSet{}, fmt.Errorf("frankfurter: upstream status %d", resp.StatusCode)
	}

	var fr frankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return ports.RateSet{}, fmt.Errorf("frankfurter: decode: %w", err)
	}

	asOf, err := time.Parse("2006-01-02", fr.Date)
	if err != nil {
		asOf = time.Now().UTC()
	}

	// Ensure base is always in the rate map with value 1.
	if fr.Rates == nil {
		fr.Rates = make(map[string]float64)
	}
	fr.Rates[base] = 1.0

	return ports.RateSet{
		Base:  fr.Base,
		AsOf:  asOf,
		Rates: fr.Rates,
	}, nil
}
