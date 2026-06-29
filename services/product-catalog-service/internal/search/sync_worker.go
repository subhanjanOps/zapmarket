package search

import (
	"context"
	"encoding/json"
	"log/slog"
)

type productEvent struct {
	EventType    string `json:"event_type"`
	ProductID    string `json:"product_id"`
	Name         string `json:"name"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	PriceCents   int64  `json:"price_cents"`
	Currency     string `json:"currency"`
	SellerID     string `json:"seller_id"`
	Status       string `json:"status"`
	ImageURL     string `json:"image_url"`
}

type SyncWorker struct {
	client *TypesenseClient
	log    *slog.Logger
}

func NewSyncWorker(client *TypesenseClient, log *slog.Logger) *SyncWorker {
	if log == nil {
		log = slog.Default()
	}
	return &SyncWorker{client: client, log: log}
}

func (w *SyncWorker) Handle(ctx context.Context, payload []byte) error {
	var evt productEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return err
	}

	switch evt.EventType {
	case "product.created", "product.updated":
		doc := SearchDoc{
			ID:           evt.ProductID,
			Name:         evt.Name,
			CategoryID:   evt.CategoryID,
			CategoryName: evt.CategoryName,
			Description:  evt.Description,
			PriceCents:   evt.PriceCents,
			Currency:     evt.Currency,
			SellerID:     evt.SellerID,
			Status:       evt.Status,
			ImageURL:     evt.ImageURL,
		}
		return w.client.IndexProduct(ctx, doc)
	case "product.deleted":
		return w.client.DeleteProduct(ctx, evt.ProductID)
	default:
		w.log.Warn("sync worker: unknown event type", "event_type", evt.EventType)
		return nil
	}
}
