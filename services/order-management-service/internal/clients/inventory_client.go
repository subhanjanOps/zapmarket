package clients

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/zapmarket/zapmarket/pkg/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InventoryClient struct {
	conn   *grpc.ClientConn
	client pb.InventoryServiceClient
}

func NewInventoryClient(addr string) (*InventoryClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial inventory-service at %s: %w", addr, err)
	}
	return &InventoryClient{conn: conn, client: pb.NewInventoryServiceClient(conn)}, nil
}

// Close drains and closes the underlying gRPC connection.
func (c *InventoryClient) Close() error {
	return c.conn.Close()
}

// ReserveStock attempts to reserve qty units of skuID for orderID.
// Returns (reservationID, true, nil) on success, (uuid.Nil, false, nil) when
// stock is insufficient, and (uuid.Nil, false, err) on transport errors.
func (c *InventoryClient) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error) {
	resp, err := c.client.ReserveStock(ctx, &pb.ReserveStockRequest{
		SkuId:    skuID.String(),
		OrderId:  orderID.String(),
		Quantity: int32(qty),
	})
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("inventory ReserveStock: %w", err)
	}
	if !resp.Reserved {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(resp.ReservationId)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("inventory returned invalid reservation_id %q: %w", resp.ReservationId, err)
	}
	return id, true, nil
}

// ReleaseStock releases a previously-created reservation.
func (c *InventoryClient) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	_, err := c.client.ReleaseStock(ctx, &pb.ReleaseStockRequest{ReservationId: reservationID.String()})
	if err != nil {
		return fmt.Errorf("inventory ReleaseStock: %w", err)
	}
	return nil
}

// DeductStock permanently removes a reservation from qty_on_hand (called
// after payment is confirmed to make the sale final).
func (c *InventoryClient) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	_, err := c.client.DeductStock(ctx, &pb.DeductStockRequest{ReservationId: reservationID.String()})
	if err != nil {
		return fmt.Errorf("inventory DeductStock: %w", err)
	}
	return nil
}
