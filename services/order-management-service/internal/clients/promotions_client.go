package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PromotionsClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPromotionsClient(baseURL string) *PromotionsClient {
	return &PromotionsClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

type ValidateCouponRequest struct {
	Code           string `json:"code"`
	UserID         string `json:"user_id"`
	CartTotalPaise int64  `json:"cart_total_paise"`
}

type ValidateCouponResult struct {
	CouponID      string `json:"coupon_id"`
	DiscountPaise int64  `json:"discount_paise"`
	FinalPaise    int64  `json:"final_paise"`
}

func (c *PromotionsClient) ValidateCoupon(ctx context.Context, req ValidateCouponRequest) (*ValidateCouponResult, error) {
	body, _ := json.Marshal(req)
	resp, err := c.httpClient.Post(c.baseURL+"/v1/coupons/validate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("promotions validate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("coupon invalid: %s", errResp.Error.Message)
	}

	var out struct {
		Data ValidateCouponResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("promotions validate decode: %w", err)
	}
	return &out.Data, nil
}

func (c *PromotionsClient) RedeemCoupon(ctx context.Context, couponID, userID, orderID string) error {
	body, _ := json.Marshal(map[string]string{"user_id": userID, "order_id": orderID})
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/v1/coupons/%s/redeem", c.baseURL, couponID),
		"application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("promotions redeem: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("promotions redeem: unexpected status %d", resp.StatusCode)
	}
	return nil
}
