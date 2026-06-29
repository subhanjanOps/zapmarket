package search

import (
	"context"
	"fmt"
	"strconv"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
)

type TypesenseAdapter struct{ client *typesense.Client }

func NewTypesenseAdapter(host, apiKey string) *TypesenseAdapter {
	c := typesense.NewClient(
		typesense.WithServer(host),
		typesense.WithAPIKey(apiKey),
	)
	return &TypesenseAdapter{client: c}
}

func (a *TypesenseAdapter) EnsureSchema(ctx context.Context) error {
	schema := &api.CollectionSchema{
		Name: "products",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "category_id", Type: "string", Facet: pointer.True()},
			{Name: "category_name", Type: "string", Facet: pointer.True()},
			{Name: "description", Type: "string"},
			{Name: "price_cents", Type: "int64"},
			{Name: "currency", Type: "string"},
			{Name: "seller_id", Type: "string"},
			{Name: "status", Type: "string", Facet: pointer.True()},
			{Name: "image_url", Type: "string", Optional: pointer.True()},
		},
		DefaultSortingField: pointer.String("price_cents"),
	}
	_, err := a.client.Collections().Create(ctx, schema)
	return err // AlreadyExists is acceptable
}

func (a *TypesenseAdapter) Upsert(ctx context.Context, doc SearchDoc) error {
	m := map[string]interface{}{
		"id":            doc.ID,
		"name":          doc.Name,
		"category_id":   doc.CategoryID,
		"category_name": doc.CategoryName,
		"description":   doc.Description,
		"price_cents":   doc.PriceCents,
		"currency":      doc.Currency,
		"seller_id":     doc.SellerID,
		"status":        doc.Status,
		"image_url":     doc.ImageURL,
	}
	_, err := a.client.Collection("products").Documents().Upsert(ctx, m)
	return err
}

func (a *TypesenseAdapter) Delete(ctx context.Context, id string) error {
	_, err := a.client.Collection("products").Document(id).Delete(ctx)
	return err
}

func (a *TypesenseAdapter) Search(ctx context.Context, q SearchQuery) ([]SearchDoc, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PerPage < 1 {
		q.PerPage = 20
	}
	queryBy := "name,description,category_name"
	params := &api.SearchCollectionParams{
		Q:       pointer.String(q.Q),
		QueryBy: pointer.String(queryBy),
		Page:    pointer.Int(q.Page),
		PerPage: pointer.Int(q.PerPage),
	}
	if q.CategoryID != "" {
		params.FilterBy = pointer.String(fmt.Sprintf("category_id:=%s", q.CategoryID))
	}
	if q.MinPrice > 0 || q.MaxPrice > 0 {
		filter := fmt.Sprintf("price_cents:>=%d && price_cents:<=%d", q.MinPrice, q.MaxPrice)
		if params.FilterBy != nil {
			filter = *params.FilterBy + " && " + filter
		}
		params.FilterBy = pointer.String(filter)
	}

	result, err := a.client.Collection("products").Documents().Search(ctx, params)
	if err != nil {
		return nil, err
	}

	var docs []SearchDoc
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document == nil {
				continue
			}
			doc := hitToDoc(*hit.Document)
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func hitToDoc(m map[string]interface{}) SearchDoc {
	str := func(key string) string {
		v, _ := m[key].(string)
		return v
	}
	i64 := func(key string) int64 {
		switch v := m[key].(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case string:
			n, _ := strconv.ParseInt(v, 10, 64)
			return n
		}
		return 0
	}
	return SearchDoc{
		ID:           str("id"),
		Name:         str("name"),
		CategoryID:   str("category_id"),
		CategoryName: str("category_name"),
		Description:  str("description"),
		PriceCents:   i64("price_cents"),
		Currency:     str("currency"),
		SellerID:     str("seller_id"),
		Status:       str("status"),
		ImageURL:     str("image_url"),
	}
}
