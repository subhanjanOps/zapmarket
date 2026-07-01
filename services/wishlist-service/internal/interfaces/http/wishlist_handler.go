package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"github.com/zapmarket/zapmarket/services/wishlist-service/internal/application/usecases"
)

type WishlistHandler struct {
	addItem    *usecases.AddToWishlistUseCase
	listItems  *usecases.ListWishlistUseCase
	removeItem *usecases.RemoveFromWishlistUseCase
	clearItems *usecases.ClearWishlistUseCase
}

func NewWishlistHandler(
	add *usecases.AddToWishlistUseCase,
	list *usecases.ListWishlistUseCase,
	remove *usecases.RemoveFromWishlistUseCase,
	clear *usecases.ClearWishlistUseCase,
) *WishlistHandler {
	return &WishlistHandler{addItem: add, listItems: list, removeItem: remove, clearItems: clear}
}

func authedUserID(r *http.Request) (string, bool) {
	claims, ok := crypto.ClaimsFromContext(r.Context())
	if !ok {
		return "", false
	}
	return claims.UserID.String(), true
}

func (h *WishlistHandler) Routes(jwtSecret string) http.Handler {
	r := chi.NewRouter()
	r.Use(crypto.RequireAuth(jwtSecret))
	r.Get("/", h.list)
	r.Post("/items", h.add)
	r.Delete("/items/{productID}", h.remove)
	r.Delete("/", h.clear)
	return r
}

func (h *WishlistHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := authedUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := h.listItems.Execute(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

type addRequest struct {
	ProductID string `json:"product_id"`
	SKUID     string `json:"sku_id"`
}

func (h *WishlistHandler) add(w http.ResponseWriter, r *http.Request) {
	userID, ok := authedUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req addRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.ProductID == "" {
		http.Error(w, "product_id is required", http.StatusBadRequest)
		return
	}
	if err := h.addItem.Execute(r.Context(), userID, req.ProductID, req.SKUID); err != nil {
		if err == usecases.ErrWishlistFull {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WishlistHandler) remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := authedUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID := chi.URLParam(r, "productID")
	if err := h.removeItem.Execute(r.Context(), userID, productID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WishlistHandler) clear(w http.ResponseWriter, r *http.Request) {
	userID, ok := authedUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.clearItems.Execute(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
