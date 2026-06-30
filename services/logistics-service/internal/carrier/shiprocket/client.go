package shiprocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/zapmarket/zapmarket/services/logistics-service/internal/application/usecases"
)

const baseURL = "https://apiv2.shiprocket.in/v1/external"

// Client implements usecases.carrierClient via Shiprocket API.
type Client struct {
	email     string
	password  string
	channelID string
	http      *http.Client

	mu        sync.Mutex
	token     string
	tokenExp  time.Time
}

func New(email, password, channelID string) *Client {
	return &Client{
		email:     email,
		password:  password,
		channelID: channelID,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateShipment calls Shiprocket adhoc order create and returns awb, carrier name, tracking URL.
func (c *Client) CreateShipment(ctx context.Context, req usecases.ShipmentRequest) (awb, carrier, trackingURL string, err error) {
	tok, err := c.token_(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("shiprocket auth: %w", err)
	}

	payload := map[string]any{
		"order_id":         req.OrderID,
		"order_date":       time.Now().Format("2006-01-02 15:04"),
		"pickup_location":  "Primary",
		"channel_id":       c.channelID,
		"billing_customer_name": "Customer",
		"billing_address":  "Address",
		"billing_city":     "City",
		"billing_pincode":  req.DeliveryPincode,
		"billing_state":    "State",
		"billing_country":  "India",
		"billing_email":    "customer@example.com",
		"billing_phone":    "9999999999",
		"shipping_is_billing": true,
		"order_items": []map[string]any{
			{"name": "Product", "sku": "SKU-1", "units": 1, "selling_price": 100},
		},
		"payment_method": "Prepaid",
		"sub_total":      100,
		"length":         10,
		"breadth":        10,
		"height":         10,
		"weight":         float64(req.WeightGrams) / 1000.0,
	}

	var result struct {
		OrderID  int    `json:"order_id"`
		Shipment struct {
			AWB        string `json:"awb_code"`
			CourierID  int    `json:"courier_company_id"`
			CourierName string `json:"courier_name"`
		} `json:"shipment"`
	}

	if err := c.post(ctx, tok, "/orders/create/adhoc", payload, &result); err != nil {
		return "", "", "", fmt.Errorf("shiprocket create shipment: %w", err)
	}

	awb = result.Shipment.AWB
	carrier = result.Shipment.CourierName
	trackingURL = fmt.Sprintf("https://shiprocket.co/tracking/%s", awb)
	return awb, carrier, trackingURL, nil
}

// CreateReversePickup creates a return pickup order with Shiprocket.
func (c *Client) CreateReversePickup(ctx context.Context, parentAWB, orderID string) (awb, carrier, trackingURL string, err error) {
	tok, err := c.token_(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("shiprocket auth: %w", err)
	}

	payload := map[string]any{
		"order_id":         orderID + "-R",
		"order_date":       time.Now().Format("2006-01-02 15:04"),
		"channel_id":       c.channelID,
		"awb_code":         parentAWB,
		"pickup_customer_name": "Customer",
		"pickup_address":   "Address",
		"pickup_city":      "City",
		"pickup_pincode":   "110001",
		"pickup_state":     "Delhi",
		"pickup_country":   "India",
		"pickup_email":     "customer@example.com",
		"pickup_phone":     "9999999999",
		"shipping_customer_name": "Warehouse",
		"shipping_address": "Warehouse Address",
		"shipping_city":    "City",
		"shipping_pincode": "110002",
		"shipping_state":   "Delhi",
		"shipping_country": "India",
		"order_items": []map[string]any{
			{"name": "Product", "sku": "SKU-1", "units": 1, "selling_price": 100},
		},
		"payment_method": "Prepaid",
		"sub_total":      100,
		"length":         10,
		"breadth":        10,
		"height":         10,
		"weight":         0.5,
	}

	var result struct {
		Shipment struct {
			AWB         string `json:"awb_code"`
			CourierName string `json:"courier_name"`
		} `json:"shipment"`
	}
	if err := c.post(ctx, tok, "/orders/create/return", payload, &result); err != nil {
		return "", "", "", fmt.Errorf("shiprocket create return: %w", err)
	}
	awb = result.Shipment.AWB
	carrier = result.Shipment.CourierName
	trackingURL = fmt.Sprintf("https://shiprocket.co/tracking/%s", awb)
	return awb, carrier, trackingURL, nil
}

// GetTrackingStatus returns the latest tracking event for an AWB.
func (c *Client) GetTrackingStatus(ctx context.Context, awb string) (status, location, description string, err error) {
	tok, err := c.token_(ctx)
	if err != nil {
		return "", "", "", err
	}

	var result struct {
		TrackingData struct {
			ShipmentTrack []struct {
				CurrentStatus string `json:"current_status"`
				Location      string `json:"location"`
			} `json:"shipment_track"`
		} `json:"tracking_data"`
	}

	if err := c.get(ctx, tok, "/courier/track/awb/"+awb, &result); err != nil {
		return "", "", "", err
	}
	if len(result.TrackingData.ShipmentTrack) > 0 {
		t := result.TrackingData.ShipmentTrack[0]
		return t.CurrentStatus, t.Location, "", nil
	}
	return "", "", "", nil
}

// token_ returns a valid JWT, refreshing if expired.
func (c *Client) token_(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}

	payload := map[string]string{"email": c.email, "password": c.password}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/login", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("shiprocket: no token in auth response")
	}
	c.token = out.Token
	c.tokenExp = time.Now().Add(23 * time.Hour) // tokens expire at 24h
	return c.token, nil
}

func (c *Client) post(ctx context.Context, tok, path string, payload, out any) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shiprocket %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(ctx context.Context, tok, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
