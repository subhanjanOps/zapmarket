package search_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/search"
)

type fakeTypesenseAPI struct {
	indexed []search.SearchDoc
	deleted []string
}

func (f *fakeTypesenseAPI) Upsert(_ context.Context, doc search.SearchDoc) error {
	f.indexed = append(f.indexed, doc)
	return nil
}

func (f *fakeTypesenseAPI) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeTypesenseAPI) Search(_ context.Context, _ search.SearchQuery) ([]search.SearchDoc, error) {
	return f.indexed, nil
}

func TestTypesenseClient_IndexProduct(t *testing.T) {
	api := &fakeTypesenseAPI{}
	client := search.NewTypesenseClient(api)

	doc := search.SearchDoc{
		ID:           "prod-1",
		Name:         "iPhone 15",
		CategoryID:   "cat-1",
		CategoryName: "Mobiles",
		Description:  "Latest Apple phone",
		PriceCents:   7999900,
		Currency:     "INR",
		SellerID:     "sel-1",
		Status:       "ACTIVE",
	}

	if err := client.IndexProduct(context.Background(), doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(api.indexed) != 1 || api.indexed[0].ID != "prod-1" {
		t.Fatal("expected product to be indexed")
	}
}
