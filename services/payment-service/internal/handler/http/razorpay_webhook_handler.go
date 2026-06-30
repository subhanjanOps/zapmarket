package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

type razorpayVerifier interface {
	VerifyWebhookSignature(body []byte, signature string) bool
}

type razorpayPaymentService interface {
	GetByGatewayTxnID(ctx context.Context, txnID string) (*domain.Payment, error)
	HandleCaptureWebhook(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string) error
}

// RazorpayWebhookHandler handles POST /webhooks/razorpay.
type RazorpayWebhookHandler struct {
	svc      razorpayPaymentService
	verifier razorpayVerifier
	logger   *slog.Logger
}

func NewRazorpayWebhookHandler(svc razorpayPaymentService, verifier razorpayVerifier, logger *slog.Logger) *RazorpayWebhookHandler {
	return &RazorpayWebhookHandler{svc: svc, verifier: verifier, logger: logger}
}

func (h *RazorpayWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Razorpay-Signature")
	if sig == "" || !h.verifier.VerifyWebhookSignature(body, sig) {
		h.logger.Warn("razorpay webhook: invalid signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity struct {
					ID      string `json:"id"`
					OrderID string `json:"order_id"`
					Status  string `json:"status"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Razorpay order_id is our GatewayTxnID (set during Charge as the Razorpay order ID).
	gatewayTxnID := event.Payload.Payment.Entity.OrderID
	switch event.Event {
	case "payment.captured":
		payment, err := h.svc.GetByGatewayTxnID(r.Context(), gatewayTxnID)
		if err != nil {
			h.logger.Error("razorpay webhook: payment not found", "txn_id", gatewayTxnID, "error", err)
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}
		if err := h.svc.HandleCaptureWebhook(r.Context(), payment.ID, gatewayTxnID); err != nil {
			h.logger.Error("razorpay webhook: capture failed", "payment_id", payment.ID, "error", err)
			http.Error(w, "capture failed", http.StatusInternalServerError)
			return
		}
		h.logger.Info("razorpay payment captured", "payment_id", payment.ID)
	case "payment.failed":
		h.logger.Warn("razorpay payment failed", "txn_id", gatewayTxnID)
	default:
		h.logger.Info("razorpay webhook: unhandled event", "event", event.Event)
	}

	w.WriteHeader(http.StatusOK)
}
