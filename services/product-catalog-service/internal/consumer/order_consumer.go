package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

type salesRankUpdater interface {
	IncrSalesRank(ctx context.Context, productID uuid.UUID) error
	UpsertCooccurrence(ctx context.Context, productA, productB uuid.UUID) error
}

// OrderConsumer listens for order.confirmed events, increments sales_rank,
// and records co-purchase pairs for recommendations.
type OrderConsumer struct {
	repo   salesRankUpdater
	logger *slog.Logger
}

func NewOrderConsumer(repo salesRankUpdater, logger *slog.Logger) *OrderConsumer {
	return &OrderConsumer{repo: repo, logger: logger}
}

func (c *OrderConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var payload struct {
		Items []struct {
			ProductID string `json:"product_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		c.logger.Warn("order consumer: bad payload", "error", err)
		return nil
	}
	var productIDs []uuid.UUID
	for _, item := range payload.Items {
		id, err := uuid.Parse(item.ProductID)
		if err != nil {
			c.logger.Warn("order consumer: invalid product_id", "product_id", item.ProductID)
			continue
		}
		if err := c.repo.IncrSalesRank(ctx, id); err != nil {
			c.logger.Error("order consumer: failed to increment sales rank", "product_id", id, "error", err)
		}
		productIDs = append(productIDs, id)
	}
	// Record co-purchase pairs.
	for i := 0; i < len(productIDs); i++ {
		for j := i + 1; j < len(productIDs); j++ {
			if err := c.repo.UpsertCooccurrence(ctx, productIDs[i], productIDs[j]); err != nil {
				c.logger.Error("order consumer: failed to upsert cooccurrence", "error", err)
			}
		}
	}
	return nil
}
