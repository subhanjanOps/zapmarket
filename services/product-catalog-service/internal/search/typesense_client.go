package search

import "context"

type SearchDoc struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	PriceCents   int64  `json:"price_cents"`
	Currency     string `json:"currency"`
	SellerID     string `json:"seller_id"`
	Status       string `json:"status"`
	ImageURL     string `json:"image_url"`
}

type SearchQuery struct {
	Q          string
	CategoryID string
	MinPrice   int64
	MaxPrice   int64
	Page       int
	PerPage    int
}

type typesenseAPI interface {
	Upsert(ctx context.Context, doc SearchDoc) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, q SearchQuery) ([]SearchDoc, error)
}

type TypesenseClient struct{ api typesenseAPI }

func NewTypesenseClient(api typesenseAPI) *TypesenseClient {
	return &TypesenseClient{api: api}
}

func (c *TypesenseClient) IndexProduct(ctx context.Context, doc SearchDoc) error {
	return c.api.Upsert(ctx, doc)
}

func (c *TypesenseClient) DeleteProduct(ctx context.Context, id string) error {
	return c.api.Delete(ctx, id)
}

func (c *TypesenseClient) Search(ctx context.Context, q SearchQuery) ([]SearchDoc, error) {
	return c.api.Search(ctx, q)
}
