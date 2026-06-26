package clients

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/zapmarket/zapmarket/pkg/proto/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func NewPaymentClient(addr string) (*PaymentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial payment-service at %s: %w", addr, err)
	}
	return &PaymentClient{conn: conn, client: pb.NewPaymentServiceClient(conn)}, nil
}

// Close drains and closes the underlying gRPC connection.
func (c *PaymentClient) Close() error {
	return c.conn.Close()
}

// ChargeCard charges the user's card for the given order. Returns the
// payment ID and the payment status string (e.g. "CAPTURED" or "FAILED").
func (c *PaymentClient) ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID) (uuid.UUID, string, error) {
	resp, err := c.client.ChargeCard(ctx, &pb.ChargeCardRequest{
		OrderId:        orderID.String(),
		UserId:         userID.String(),
		Amount:         amount,
		Currency:       currency,
		IdempotencyKey: idempotencyKey.String(),
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("payment ChargeCard: %w", err)
	}
	paymentID, err := uuid.Parse(resp.PaymentId)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("payment returned invalid payment_id %q: %w", resp.PaymentId, err)
	}
	return paymentID, resp.Status, nil
}
