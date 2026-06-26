package clients

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/zapmarket/zapmarket/pkg/proto/catalog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type CatalogClient struct {
	conn   *grpc.ClientConn
	client pb.ProductCatalogServiceClient
}

func NewCatalogClient(addr string) (*CatalogClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial catalog-service at %s: %w", addr, err)
	}
	return &CatalogClient{conn: conn, client: pb.NewProductCatalogServiceClient(conn)}, nil
}

func (c *CatalogClient) Close() error {
	return c.conn.Close()
}

// GetSKUPrice returns the authoritative price (in cents) for the given SKU.
// Returns an error if the SKU does not exist or is not active.
func (c *CatalogClient) GetSKUPrice(ctx context.Context, skuID uuid.UUID) (int64, error) {
	resp, err := c.client.GetSKU(ctx, &pb.GetSKURequest{SkuId: skuID.String()})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return 0, fmt.Errorf("SKU %s not found", skuID)
		}
		return 0, fmt.Errorf("catalog GetSKU: %w", err)
	}
	if resp.GetSku() == nil {
		return 0, fmt.Errorf("catalog returned nil SKU for %s", skuID)
	}
	return resp.GetSku().GetPriceAmount(), nil
}
