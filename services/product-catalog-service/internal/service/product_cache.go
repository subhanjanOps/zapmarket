package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
)

const productCacheTTL = 5 * time.Minute

type productListResult struct {
	Products []*domain.Product `json:"products"`
	Total    int64             `json:"total"`
}

// cachedProductService wraps ProductService with a Redis cache layer.
// Cache-aside: read from Redis first; on miss, read from DB and populate.
// Mutations (Create/Update/Delete) delete the product's own keys.
// List pages use a short TTL and are not explicitly invalidated on mutation —
// 5 minutes of stale list data is acceptable for a catalog service.
type cachedProductService struct {
	inner  ProductService
	rdb    *goredis.Client
	logger *slog.Logger
}

// NewCachedProductService wraps an existing ProductService with Redis caching.
func NewCachedProductService(inner ProductService, rdb *goredis.Client, logger *slog.Logger) ProductService {
	return &cachedProductService{inner: inner, rdb: rdb, logger: logger}
}

func productIDKey(id uuid.UUID) string   { return fmt.Sprintf("product:id:%s", id) }
func productSlugKey(slug string) string  { return fmt.Sprintf("product:slug:%s", slug) }
func productListKey(filters *domain.ProductFilters) string {
	b, err := json.Marshal(filters)
	if err != nil {
		return fmt.Sprintf("product:list:err:%p", filters)
	}
	return fmt.Sprintf("product:list:%x", md5.Sum(b))
}

func (c *cachedProductService) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	key := productIDKey(id)
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var p domain.Product
		if json.Unmarshal(b, &p) == nil {
			return &p, nil
		}
	}
	p, err := c.inner.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(p); err == nil {
		_ = c.rdb.Set(ctx, key, b, productCacheTTL).Err()
		// Also populate slug key so both lookups share one DB read.
		_ = c.rdb.Set(ctx, productSlugKey(p.Slug), b, productCacheTTL).Err()
	}
	return p, nil
}

func (c *cachedProductService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	key := productSlugKey(slug)
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var p domain.Product
		if json.Unmarshal(b, &p) == nil {
			return &p, nil
		}
	}
	p, err := c.inner.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(p); err == nil {
		_ = c.rdb.Set(ctx, key, b, productCacheTTL).Err()
		_ = c.rdb.Set(ctx, productIDKey(p.ID), b, productCacheTTL).Err()
	}
	return p, nil
}

func (c *cachedProductService) GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, int64, error) {
	key := productListKey(filters)
	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var r productListResult
		if json.Unmarshal(b, &r) == nil {
			return r.Products, r.Total, nil
		}
	}
	products, total, err := c.inner.GetProductList(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	if b, err := json.Marshal(productListResult{Products: products, Total: total}); err == nil {
		_ = c.rdb.Set(ctx, key, b, productCacheTTL).Err()
	}
	return products, total, nil
}

func (c *cachedProductService) CreateProduct(ctx context.Context, product *domain.Product) error {
	if err := c.inner.CreateProduct(ctx, product); err != nil {
		return err
	}
	// List pages are short-TTL; only invalidate ID/slug keys on creation.
	return nil
}

func (c *cachedProductService) UpdateProduct(ctx context.Context, product *domain.Product) error {
	if err := c.inner.UpdateProduct(ctx, product); err != nil {
		return err
	}
	_ = c.rdb.Del(ctx, productIDKey(product.ID), productSlugKey(product.Slug)).Err()
	return nil
}

func (c *cachedProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// Fetch slug before deleting so we can clear the slug key.
	p, _ := c.inner.GetProductByID(ctx, id)
	if err := c.inner.DeleteProduct(ctx, id); err != nil {
		return err
	}
	keys := []string{productIDKey(id)}
	if p != nil {
		keys = append(keys, productSlugKey(p.Slug))
	}
	_ = c.rdb.Del(ctx, keys...).Err()
	return nil
}
