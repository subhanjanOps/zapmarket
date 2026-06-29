package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/search"
)

type searcher interface {
	Search(ctx context.Context, q search.SearchQuery) ([]search.SearchDoc, error)
}

type SearchHandler struct{ searcher searcher }

func NewSearchHandler(s searcher) *SearchHandler { return &SearchHandler{searcher: s} }

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	minPrice, _ := strconv.ParseInt(q.Get("min_price"), 10, 64)
	maxPrice, _ := strconv.ParseInt(q.Get("max_price"), 10, 64)
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	results, err := h.searcher.Search(r.Context(), search.SearchQuery{
		Q:          q.Get("q"),
		CategoryID: q.Get("category_id"),
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results, "page": page, "per_page": perPage})
}
