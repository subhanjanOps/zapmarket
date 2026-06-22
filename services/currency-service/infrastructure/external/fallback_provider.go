package external

import (
	"context"
	"fmt"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

// FallbackProvider tries the primary provider first, then the secondary on error.
type FallbackProvider struct {
	primary   ports.RatesProvider
	secondary ports.RatesProvider
}

func NewFallbackProvider(primary, secondary ports.RatesProvider) *FallbackProvider {
	return &FallbackProvider{primary: primary, secondary: secondary}
}

func (f *FallbackProvider) FetchLatest(ctx context.Context, base string) (ports.RateSet, error) {
	rs, err := f.primary.FetchLatest(ctx, base)
	if err == nil {
		return rs, nil
	}
	rs2, err2 := f.secondary.FetchLatest(ctx, base)
	if err2 != nil {
		return ports.RateSet{}, fmt.Errorf("both providers failed: primary=%w; secondary=%v", err, err2)
	}
	return rs2, nil
}
