// Package consumer handles Kafka events and dispatches notifications.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/notifier"
)

// Handler processes order events from Kafka and dispatches notifications.
type Handler struct {
	notifier notifier.Notifier
	redis    *redis.Client
	logger   *slog.Logger
}

func New(n notifier.Notifier, rdb *redis.Client, logger *slog.Logger) *Handler {
	return &Handler{notifier: n, redis: rdb, logger: logger}
}

// Handle is a kafka.HandlerFunc compatible method.
func (h *Handler) Handle(ctx context.Context, msg pkgkafka.Message) error {
	eventType := msg.Headers["event_type"]
	outboxID := msg.Headers["outbox_id"]

	// Dedup: skip if we already processed this outbox event.
	dedupKey := fmt.Sprintf("notif:dedup:%s", outboxID)
	set, err := h.redis.SetNX(ctx, dedupKey, 1, time.Hour).Result()
	if err != nil {
		h.logger.Error("redis dedup check failed", "outbox_id", outboxID, "error", err)
		// Continue processing rather than blocking on Redis errors.
	} else if !set {
		h.logger.Info("duplicate event skipped", "outbox_id", outboxID, "event_type", eventType)
		return nil
	}

	var payload map[string]string
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Error("failed to parse event payload", "event_type", eventType, "error", err)
		return nil // Don't retry malformed messages.
	}

	notif, ok := h.buildNotification(eventType, payload)
	if !ok {
		h.logger.Info("no notification template for event", "event_type", eventType)
		return nil
	}

	if err := h.notifier.Send(ctx, notif); err != nil {
		h.logger.Error("failed to send notification", "event_type", eventType, "user_id", notif.UserID, "error", err)
		return err // Retry.
	}

	h.logger.Info("notification sent", "event_type", eventType, "user_id", notif.UserID)
	return nil
}

func (h *Handler) buildNotification(eventType string, payload map[string]string) (notifier.Notification, bool) {
	switch eventType {
	case "order.confirmed":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Your order has been confirmed!",
			Body:      fmt.Sprintf("Order %s has been confirmed and payment captured. Thank you for shopping with ZapMarket!", payload["order_id"]),
		}, true

	case "order.cancelled":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Your order has been cancelled",
			Body:      fmt.Sprintf("Order %s has been cancelled. If you have any questions, please contact support.", payload["order_id"]),
		}, true

	default:
		return notifier.Notification{}, false
	}
}
