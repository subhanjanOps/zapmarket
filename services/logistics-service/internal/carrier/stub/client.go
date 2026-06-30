package stub

import (
	"context"
	"fmt"
	"time"

	"github.com/zapmarket/zapmarket/services/logistics-service/internal/application/usecases"
)

// Client is a no-op carrier for local dev and tests.
type Client struct{}

func New() *Client { return &Client{} }

func (c *Client) CreateShipment(_ context.Context, req usecases.ShipmentRequest) (awb, carrier, trackingURL string, err error) {
	awb = fmt.Sprintf("STUB-%d", time.Now().UnixMilli())
	return awb, "stub-carrier", "https://example.com/track/" + awb, nil
}

func (c *Client) CreateReversePickup(_ context.Context, parentAWB, _ string) (awb, carrier, trackingURL string, err error) {
	awb = fmt.Sprintf("STUB-R-%d", time.Now().UnixMilli())
	return awb, "stub-carrier", "https://example.com/track/" + awb, nil
}
