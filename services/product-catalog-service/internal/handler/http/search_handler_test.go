package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/search"
)

type fakeSearcher struct{ results []search.SearchDoc }

func (f *fakeSearcher) Search(_ context.Context, _ search.SearchQuery) ([]search.SearchDoc, error) {
	return f.results, nil
}

func TestSearchHandler_Returns200WithResults(t *testing.T) {
	searcher := &fakeSearcher{results: []search.SearchDoc{{ID: "p1", Name: "iPhone"}}}
	h := handler.NewSearchHandler(searcher)

	req := httptest.NewRequest(http.MethodGet, "/v1/products/search?q=iphone", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
