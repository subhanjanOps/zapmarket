package grpc

import (
	"context"

	"google.golang.org/grpc"
	pb "github.com/zapmarket/zapmarket/pkg/proto/catalog"
)

// CatalogSKUFetcher fetches current SKU price from product-catalog-service via gRPC.
type CatalogSKUFetcher struct {
	client pb.ProductCatalogServiceClient
}

func NewCatalogSKUFetcher(conn *grpc.ClientConn) *CatalogSKUFetcher {
	return &CatalogSKUFetcher{client: pb.NewProductCatalogServiceClient(conn)}
}

func (f *CatalogSKUFetcher) GetSKUPrice(ctx context.Context, skuID string) (priceCents int64, currency string, inStock bool, err error) {
	resp, err := f.client.GetSKU(ctx, &pb.GetSKURequest{SkuId: skuID})
	if err != nil {
		return 0, "", false, err
	}
	sku := resp.GetSku()
	if sku == nil {
		return 0, "", false, nil
	}
	// Proto doesn't expose stock level — optimistically assume in-stock;
	// actual availability is enforced at reservation time by inventory-service.
	return sku.PriceAmount, sku.Currency, true, nil
}
