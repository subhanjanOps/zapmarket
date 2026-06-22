package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ratesKey = "fx:rates:USD"
	rateTTL  = time.Hour
	source   = "https://api.frankfurter.app/latest?from=USD"
)

type RatesResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// Fetch returns current USD-base exchange rates, serving from Redis cache when
// available and falling back to a live call to frankfurter.app otherwise.
func Fetch(ctx context.Context, rdb *redis.Client) (*RatesResponse, error) {
	if raw, err := rdb.Get(ctx, ratesKey).Bytes(); err == nil {
		var cached RatesResponse
		if json.Unmarshal(raw, &cached) == nil {
			return &cached, nil
		}
	}

	resp, err := fetchLive(ctx)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(resp); err == nil {
		_ = rdb.Set(ctx, ratesKey, b, rateTTL).Err()
	}
	return resp, nil
}

func fetchLive(ctx context.Context) (*RatesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("currency fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("currency fetch: upstream status %d", resp.StatusCode)
	}

	var r RatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("currency fetch: decode: %w", err)
	}
	// Always include USD itself so callers have a complete map.
	if r.Rates == nil {
		r.Rates = make(map[string]float64)
	}
	r.Rates["USD"] = 1
	return &r, nil
}
