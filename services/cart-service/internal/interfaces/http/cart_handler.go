package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type CartHandler struct {
	addItem    *usecases.AddItemUseCase
	removeItem *usecases.RemoveItemUseCase
	getCart    *usecases.GetCartUseCase
	mergeCart  *usecases.MergeCartUseCase
}

func NewCartHandler(
	add *usecases.AddItemUseCase,
	remove *usecases.RemoveItemUseCase,
	get *usecases.GetCartUseCase,
	merge *usecases.MergeCartUseCase,
) *CartHandler {
	return &CartHandler{addItem: add, removeItem: remove, getCart: get, mergeCart: merge}
}

func (h *CartHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.getCartHandler)
	r.Post("/items", h.addItemHandler)
	r.Delete("/items/{skuID}", h.removeItemHandler)
	r.Post("/merge", h.mergeHandler)
	return r
}

func (h *CartHandler) getCartHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	cart, err := h.getCart.Execute(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *CartHandler) addItemHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	var item domain.CartItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.addItem.Execute(r.Context(), userID, item); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) removeItemHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	skuID := chi.URLParam(r, "skuID")
	if err := h.removeItem.Execute(r.Context(), userID, skuID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) mergeHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	sessionID := r.Header.Get("X-Session-ID")
	if err := h.mergeCart.Execute(r.Context(), sessionID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
