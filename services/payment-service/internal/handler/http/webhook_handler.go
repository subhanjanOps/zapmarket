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

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/service"
)

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
	Status        string `json:"status"` // "captured" or "failed"
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
	case "captured":
		err = h.svc.HandleCaptureWebhook(r.Context(), paymentID, payload.GatewayTxnID)
	case "failed":
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
