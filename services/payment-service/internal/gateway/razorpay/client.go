// Package razorpay implements contracts.PaymentGateway for Razorpay (India).
package razorpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"
)

const baseURL = "https://api.razorpay.com/v1"

// Gateway implements contracts.PaymentGateway using Razorpay.
type Gateway struct {
	keyID     string
	keySecret string
	webhookSecret string
	http      *http.Client
}

func New(keyID, keySecret, webhookSecret string) *Gateway {
	return &Gateway{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *Gateway) Name() string { return "razorpay" }

// Charge creates a Razorpay Order and returns the order_id as GatewayTxnID.
// The actual charge happens on the frontend via Razorpay Checkout JS after
// the frontend receives the order_id, followed by a webhook confirmation.
func (g *Gateway) Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*contracts.ChargeResult, error) {
	payload := map[string]any{
		"amount":   amount,
		"currency": currency,
		"receipt":  idempotencyKey.String(),
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := g.post(ctx, "/orders", payload, &result); err != nil {
		return nil, pkgerrors.NewInternal("PAYMENT_ERROR", "razorpay create order failed: "+err.Error(), err)
	}
	return &contracts.ChargeResult{GatewayTxnID: result.ID}, nil
}

// Capture calls POST /v1/payments/{id}/capture for server-side capture.
func (g *Gateway) Capture(ctx context.Context, paymentID string, amount int64, currency string) error {
	payload := map[string]any{"amount": amount, "currency": currency}
	var result map[string]any
	return g.post(ctx, "/payments/"+paymentID+"/capture", payload, &result)
}

// Refund creates a Razorpay refund against a payment.
func (g *Gateway) Refund(ctx context.Context, gatewayTxnID string, amount int64, currency string) (string, error) {
	payload := map[string]any{"amount": amount}
	var result struct {
		ID string `json:"id"`
	}
	if err := g.post(ctx, "/payments/"+gatewayTxnID+"/refund", payload, &result); err != nil {
		return "", pkgerrors.NewInternal("REFUND_ERROR", "razorpay refund failed: "+err.Error(), err)
	}
	return result.ID, nil
}

// VerifyWebhookSignature returns true if the X-Razorpay-Signature matches.
func (g *Gateway) VerifyWebhookSignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(g.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (g *Gateway) post(ctx context.Context, path string, payload, out any) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(g.keyID, g.keySecret)

	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
