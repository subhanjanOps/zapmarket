package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

const keyPrefix = "currency:rates:"

type RatesCache struct{ rdb *goredis.Client }

func NewRatesCache(rdb *goredis.Client) *RatesCache { return &RatesCache{rdb: rdb} }

func (c *RatesCache) Get(ctx context.Context, base string) (ports.CachedRates, bool, error) {
	raw, err := c.rdb.Get(ctx, keyPrefix+base).Bytes()
	if err == goredis.Nil {
		return ports.CachedRates{}, false, nil
	}
	if err != nil {
		return ports.CachedRates{}, false, fmt.Errorf("cache get: %w", err)
	}
	var cached ports.CachedRates
	if err := json.Unmarshal(raw, &cached); err != nil {
		return ports.CachedRates{}, false, fmt.Errorf("cache unmarshal: %w", err)
	}
	return cached, true, nil
}

func (c *RatesCache) Set(ctx context.Context, base string, r ports.CachedRates, ttl time.Duration) error {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	if err := c.rdb.Set(ctx, keyPrefix+base, b, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}
