// Package http holds payment-service's HTTP surface: a health check and
// the payment gateway's async webhook callback. Everything else is gRPC
// (see design.md) — Payment has no public REST API for clients.
package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/service"
)

// webhookMaxAge is the maximum age of a webhook request. Requests older than
// this are rejected to prevent replay attacks.
const webhookMaxAge = 5 * time.Minute

// WebhookHandler verifies and dispatches the payment gateway's async
// status callbacks. Built ahead of any real gateway integration (see
// planning/04-payment-service.md) — FakePaymentGateway never calls this,
// but the signature-verification logic is generic HMAC-SHA256-over-body,
// the same scheme Razorpay/Stripe-style webhooks use, so it's exercisable
// and testable now and should need no rework when a real gateway is wired.
type WebhookHandler struct {
	svc    service.PaymentService
	secret string
	logger *slog.Logger
}

func NewWebhookHandler(svc service.PaymentService, secret string, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, secret: secret, logger: logger}
}

type webhookPayload struct {
	PaymentID     string `json:"payment_id"`
	Status        string `json:"status"` // "CAPTURED" or "FAILED"
	GatewayTxnID  string `json:"gateway_txn_id,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
}

// Health responds 200 if the process is up.
func (h *WebhookHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// HandlePaymentWebhook verifies the X-Webhook-Signature header (hex-encoded
// HMAC-SHA256 of the raw body using the configured secret) before trusting
// anything in the payload — an unsigned or wrongly-signed request is
// rejected outright.
func (h *WebhookHandler) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	// Reject stale requests before reading the body.
	tsHeader := r.Header.Get("X-Webhook-Timestamp")
	if tsHeader == "" {
		http.Error(w, "missing X-Webhook-Timestamp", http.StatusBadRequest)
		return
	}
	tsUnix, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil || time.Since(time.Unix(tsUnix, 0)).Abs() > webhookMaxAge {
		http.Error(w, "webhook timestamp out of range", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB cap
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if !h.validSignature(body, r.Header.Get("X-Webhook-Signature")) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	paymentID, err := uuid.Parse(payload.PaymentID)
	if err != nil {
		http.Error(w, "invalid payment_id", http.StatusBadRequest)
		return
	}

	switch payload.Status {
	case "CAPTURED":
		err = h.svc.HandleCaptureWebhook(r.Context(), paymentID, payload.GatewayTxnID)
	case "FAILED":
		err = h.svc.HandleFailureWebhook(r.Context(), paymentID, payload.FailureReason)
	default:
		http.Error(w, "unrecognized status", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("failed to process payment webhook", "payment_id", paymentID, "error", err)
		http.Error(w, "failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) validSignature(body []byte, signatureHeader string) bool {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

// HandleStripeWebhook verifies Stripe-Signature and dispatches payment_intent events.
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.secret)
	if err != nil {
		h.logger.Warn("stripe webhook signature verification failed", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	switch event.Type {
	case stripe.EventTypePaymentIntentSucceeded:
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			http.Error(w, "invalid event data", http.StatusBadRequest)
			return
		}
		paymentID, parseErr := uuid.Parse(pi.Metadata["payment_id"])
		if parseErr != nil {
			h.logger.Warn("stripe webhook: missing payment_id in metadata", "pi_id", pi.ID)
			w.WriteHeader(http.StatusOK) // ack so Stripe doesn't retry
			return
		}
		if err := h.svc.HandleCaptureWebhook(r.Context(), paymentID, pi.ID); err != nil {
			h.logger.Error("failed to handle stripe capture", "payment_id", paymentID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

	case stripe.EventTypePaymentIntentPaymentFailed:
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			http.Error(w, "invalid event data", http.StatusBadRequest)
			return
		}
		paymentID, parseErr := uuid.Parse(pi.Metadata["payment_id"])
		if parseErr != nil {
			h.logger.Warn("stripe webhook: missing payment_id in metadata", "pi_id", pi.ID)
			w.WriteHeader(http.StatusOK)
			return
		}
		reason := "payment failed"
		if pi.LastPaymentError != nil {
			reason = pi.LastPaymentError.Msg
		}
		if err := h.svc.HandleFailureWebhook(r.Context(), paymentID, reason); err != nil {
			h.logger.Error("failed to handle stripe failure", "payment_id", paymentID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

	default:
		// Acknowledge unhandled event types so Stripe doesn't retry them.
	}

	w.WriteHeader(http.StatusOK)
}
