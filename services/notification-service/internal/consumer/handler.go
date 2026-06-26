// Package consumer handles Kafka events and dispatches notifications.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/notifier"
)

// deduplicator abstracts the Redis operations used for notification dedup.
type deduplicator interface {
	// SetNX claims the dedup key atomically. Returns true if the claim was acquired.
	SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Del removes the dedup key (used to release the claim on send failure).
	Del(ctx context.Context, key string) error
}

var zeroDecimalCurrencies = map[string]bool{
	"JPY": true, "KRW": true, "IDR": true,
}

var currencySymbols = map[string]string{
	"USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "KRW": "₩",
	"INR": "₹", "CNY": "¥", "AUD": "A$", "CAD": "C$", "CHF": "Fr",
}

func formatAmount(amountCents string, currency string) string {
	cents, err := strconv.ParseInt(amountCents, 10, 64)
	if err != nil {
		return amountCents + " " + currency
	}
	sym := currencySymbols[currency]
	if sym == "" {
		sym = currency + " "
	}
	if zeroDecimalCurrencies[currency] {
		return sym + formatWithCommas(cents)
	}
	whole := cents / 100
	frac := cents % 100
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%s%s.%02d", sym, formatWithCommas(whole), frac)
}

func formatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if i > 0 || rem > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// Handler processes order events from Kafka and dispatches notifications.
type Handler struct {
	notifier notifier.Notifier
	dedup    deduplicator
	logger   *slog.Logger
}

func New(n notifier.Notifier, dedup deduplicator, logger *slog.Logger) *Handler {
	return &Handler{notifier: n, dedup: dedup, logger: logger}
}

// Handle is a kafka.HandlerFunc compatible method.
func (h *Handler) Handle(ctx context.Context, msg pkgkafka.Message) error {
	eventType := msg.Headers["event_type"]
	outboxID := msg.Headers["outbox_id"]

	if outboxID == "" || eventType == "" {
		return fmt.Errorf("missing required headers: outbox_id=%q event_type=%q", outboxID, eventType)
	}

	// H3: validate outbox_id is a valid UUID before using it as a dedup key.
	dedupEnabled := true
	if _, err := uuid.Parse(outboxID); err != nil {
		h.logger.Warn("outbox_id is not a valid UUID, skipping dedup", "outbox_id", outboxID)
		dedupEnabled = false
	}

	dedupKey := fmt.Sprintf("notif:dedup:%s", outboxID)

	// Claim the dedup key atomically before sending. SetNX is the guard — if
	// another consumer already claimed the key we skip silently.
	if dedupEnabled {
		claimed, err := h.dedup.SetNX(ctx, dedupKey, 72*time.Hour)
		if err != nil {
			return fmt.Errorf("dedup SetNX: %w", err)
		}
		if !claimed {
			h.logger.Info("duplicate event skipped", "outbox_id", outboxID, "event_type", eventType)
			return nil
		}
	}

	// Parse into any-typed map first to handle numeric/boolean fields in
	// payloads (e.g. inventory.reserved publishes qty as a JSON number).
	var raw map[string]any
	if err := json.Unmarshal(msg.Value, &raw); err != nil {
		h.logger.Error("failed to parse event payload", "event_type", eventType, "error", err)
		return nil // Don't retry malformed messages.
	}
	// Coerce every field to string so notification templates can use a uniform type.
	payload := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			payload[k] = val
		case nil:
			payload[k] = ""
		default:
			payload[k] = fmt.Sprintf("%v", val)
		}
	}

	notif, ok := h.buildNotification(eventType, payload)
	if !ok {
		h.logger.Info("no notification template for event", "event_type", eventType)
		return nil
	}

	if err := h.notifier.Send(ctx, notif); err != nil {
		h.logger.Error("failed to send notification", "event_type", eventType, "user_id", notif.UserID, "error", err)
		// Release the dedup claim so the next retry can reclaim it.
		if dedupEnabled {
			if delErr := h.dedup.Del(ctx, dedupKey); delErr != nil {
				h.logger.Warn("failed to release dedup key after send failure", "outbox_id", outboxID, "error", delErr)
			}
		}
		return err // Retry.
	}

	h.logger.Info("notification sent", "event_type", eventType, "user_id", notif.UserID)
	return nil
}

func (h *Handler) buildNotification(eventType string, payload map[string]string) (notifier.Notification, bool) {
	switch eventType {

	// ── Order events ──────────────────────────────────────────────────────────
	case "order.confirmed":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Your order has been confirmed",
			Body:      fmt.Sprintf("Order %s has been confirmed and payment captured. Thank you for shopping with ZapMarket!", payload["order_id"]),
		}, true

	case "order.cancelled":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Your order has been cancelled",
			Body:      fmt.Sprintf("Order %s has been cancelled. If you have any questions, please contact support.", payload["order_id"]),
		}, true

	// ── Payment events ────────────────────────────────────────────────────────
	case "payment.captured":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Payment successful",
			Body:      fmt.Sprintf("Your payment of %s for order %s was successful.", formatAmount(payload["amount"], payload["currency"]), payload["order_id"]),
		}, true

	case "payment.failed":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Payment failed",
			Body:      fmt.Sprintf("We were unable to process your payment for order %s. Please update your payment details and try again.", payload["order_id"]),
		}, true

	case "payment.refunded":
		return notifier.Notification{
			UserID:    payload["user_id"],
			EventType: eventType,
			Subject:   "Refund processed",
			Body:      fmt.Sprintf("A refund of %s for order %s has been processed and will appear within 3-5 business days.", formatAmount(payload["amount"], payload["currency"]), payload["order_id"]),
		}, true

	// ── Inventory events ──────────────────────────────────────────────────────
	case "inventory.reserved":
		// Informational — no user-facing notification needed.
		return notifier.Notification{}, false

	case "inventory.depleted":
		if payload["seller_id"] == "" {
			return notifier.Notification{}, false
		}
		return notifier.Notification{
			UserID:    payload["seller_id"],
			EventType: eventType,
			Subject:   "Stock depleted for your product",
			Body:      fmt.Sprintf("SKU %s is now out of stock. Update your inventory to continue selling.", payload["sku_id"]),
		}, true

	default:
		return notifier.Notification{}, false
	}
}
