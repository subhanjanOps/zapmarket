package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const razorpayBase = "https://api.razorpay.com/v1"

// PayoutGateway calls the Razorpay Payouts API to disburse seller funds.
type PayoutGateway struct {
	keyID     string
	keySecret string
	accountID string // Razorpay X account number
	http      *http.Client
}

func NewPayoutGateway(keyID, keySecret, accountID string) *PayoutGateway {
	return &PayoutGateway{
		keyID:     keyID,
		keySecret: keySecret,
		accountID: accountID,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// Initiate creates a Razorpay payout and returns the payout ID.
// sellerID is stored as notes for reconciliation.
func (g *PayoutGateway) Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (string, error) {
	payload := map[string]any{
		"account_number": g.accountID,
		"fund_account_id": sellerID, // caller should pass actual fund_account_id
		"amount":          amountPaise,
		"currency":        currency,
		"mode":            "NEFT",
		"purpose":         "payout",
		"queue_if_low_balance": true,
		"reference_id":    fmt.Sprintf("settlement-%s-%d", sellerID, time.Now().Unix()),
		"notes":           map[string]string{"seller_id": sellerID},
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, razorpayBase+"/payouts", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(g.keyID, g.keySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("razorpay payout: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.ID, nil
}
