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
	groupFn := formatWithCommasWestern
	if currency == "INR" {
		groupFn = formatWithCommasINR
	}
	if zeroDecimalCurrencies[currency] {
		return sym + groupFn(cents)
	}
	whole := cents / 100
	frac := cents % 100
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%s%s.%02d", sym, groupFn(whole), frac)
}

// formatWithCommas formats n with South Asian grouping (3-2-2-... from the right)
// for INR, and standard Western grouping (3-3-3) for all other currencies.
// The currency parameter is passed through from formatAmount.
func formatWithCommasWestern(n int64) string {
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

// formatWithCommasINR formats n using South Asian grouping: first group is 3
// digits from the right, then groups of 2 (e.g. 1,23,45,678).
func formatWithCommasINR(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	// Last 3 digits form the first (rightmost) group.
	tail := s[len(s)-3:]
	head := s[:len(s)-3]
	// Remaining digits are grouped in 2s from the right.
	rem := len(head) % 2
	if rem > 0 {
		b.WriteString(head[:rem])
	}
	for i := rem; i < len(head); i += 2 {
		if i > 0 || rem > 0 {
			b.WriteByte(',')
		}
		b.WriteString(head[i : i+2])
	}
	b.WriteByte(',')
	b.WriteString(tail)
	return b.String()
}

// Handler processes order events from Kafka and dispatches notifications.
type Handler struct {
	notifier notifier.Notifier
	sms      notifier.SMSNotifier
	dedup    deduplicator
	logger   *slog.Logger
}

func New(n notifier.Notifier, dedup deduplicator, logger *slog.Logger) *Handler {
	return &Handler{notifier: n, dedup: dedup, logger: logger}
}

func NewWithSMS(n notifier.Notifier, sms notifier.SMSNotifier, dedup deduplicator, logger *slog.Logger) *Handler {
	return &Handler{notifier: n, sms: sms, dedup: dedup, logger: logger}
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

	// Fire-and-forget SMS for order events when phone is in the payload.
	if h.sms != nil && payload["phone"] != "" && (eventType == "order.confirmed" || eventType == "order.cancelled") {
		phone := payload["phone"]
		smsBody := notif.Body
		go func() {
			if err := h.sms.SendSMS(ctx, phone, smsBody); err != nil {
				h.logger.Warn("SMS notification failed", "event_type", eventType, "error", err)
			}
		}()
	}

	if err := h.notifier.Send(ctx, notif); err != nil {
		h.logger.Error("failed to send notification", "event_type", eventType, "user_id", notif.UserID, "error", err)
		// Do NOT release the dedup claim. If the notifier partially succeeded
		// (e.g. SMTP accepted the message but then returned a transient error),
		// releasing the claim would cause a duplicate delivery on the next Kafka
		// retry. We prefer potential message loss over guaranteed double-delivery.
		// The Kafka consumer will retry; if the dedup key is still held those
		// retries will be skipped. Operators can manually delete the Redis key
		// to force a re-send if needed.
		return err
	}

	h.logger.Info("notification sent", "event_type", eventType, "user_id", notif.UserID)
	return nil
}

// notifTemplate describes how to build a notification for one event type.
// userIDField names the payload key that holds the recipient user ID.
type notifTemplate struct {
	userIDField string
	subject     string
	body        func(payload map[string]string) string
}

// notifTemplates is the source of truth for all notification copy.
// Add or modify entries here without touching handler logic.
var notifTemplates = map[string]notifTemplate{
	"order.confirmed": {
		userIDField: "user_id",
		subject:     "Your order has been confirmed",
		body: func(p map[string]string) string {
			return fmt.Sprintf("Order %s has been confirmed and payment captured. Thank you for shopping with ZapMarket!", p["order_id"])
		},
	},
	"order.cancelled": {
		userIDField: "user_id",
		subject:     "Your order has been cancelled",
		body: func(p map[string]string) string {
			return fmt.Sprintf("Order %s has been cancelled. If you have any questions, please contact support.", p["order_id"])
		},
	},
	"payment.captured": {
		userIDField: "user_id",
		subject:     "Payment successful",
		body: func(p map[string]string) string {
			return fmt.Sprintf("Your payment of %s for order %s was successful.", formatAmount(p["amount"], p["currency"]), p["order_id"])
		},
	},
	"payment.failed": {
		userIDField: "user_id",
		subject:     "Payment failed",
		body: func(p map[string]string) string {
			return fmt.Sprintf("We were unable to process your payment for order %s. Please update your payment details and try again.", p["order_id"])
		},
	},
	"payment.refunded": {
		userIDField: "user_id",
		subject:     "Refund processed",
		body: func(p map[string]string) string {
			return fmt.Sprintf("A refund of %s for order %s has been processed and will appear within 3-5 business days.", formatAmount(p["amount"], p["currency"]), p["order_id"])
		},
	},
	"inventory.depleted": {
		userIDField: "seller_id",
		subject:     "Stock depleted for your product",
		body: func(p map[string]string) string {
			return fmt.Sprintf("SKU %s is now out of stock. Update your inventory to continue selling.", p["sku_id"])
		},
	},
}

func (h *Handler) buildNotification(eventType string, payload map[string]string) (notifier.Notification, bool) {
	tmpl, ok := notifTemplates[eventType]
	if !ok {
		return notifier.Notification{}, false
	}
	userID := payload[tmpl.userIDField]
	if userID == "" {
		return notifier.Notification{}, false
	}
	return notifier.Notification{
		UserID:    userID,
		EventType: eventType,
		Subject:   tmpl.subject,
		Body:      tmpl.body(payload),
	}, true
}
