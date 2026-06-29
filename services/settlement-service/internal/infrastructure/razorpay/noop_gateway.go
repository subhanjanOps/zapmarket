package razorpay

import (
	"context"
	"log/slog"
)

type NoopGateway struct{ log *slog.Logger }

func NewNoopGateway(log *slog.Logger) *NoopGateway {
	if log == nil {
		log = slog.Default()
	}
	return &NoopGateway{log: log}
}

func (g *NoopGateway) Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (string, error) {
	g.log.InfoContext(ctx, "noop payout gateway: would initiate payout",
		"seller_id", sellerID, "amount_paise", amountPaise, "currency", currency)
	return "noop-payout-id", nil
}
