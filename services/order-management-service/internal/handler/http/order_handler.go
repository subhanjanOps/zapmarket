package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/authctx"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/service"
)

type orderResponse struct {
	*domain.Order
	Items []*domain.OrderItem `json:"items,omitempty"`
}

type OrderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type checkoutItemRequest struct {
	SKUID     string `json:"sku_id"`
	SellerID  string `json:"seller_id,omitempty"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

type checkoutRequest struct {
	Items          []checkoutItemRequest `json:"items"`
	IdempotencyKey string                `json:"idempotency_key"`
	Currency       string                `json:"currency,omitempty"`
}

// Checkout handles POST /v1/orders.
//
//	@Summary		Place an order
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		checkoutRequest	true	"Checkout payload"
//	@Success		201		{object}	Response{data=domain.Order}
//	@Failure		400		{object}	Response
//	@Failure		401		{object}	Response
//	@Failure		409		{object}	Response
//	@Failure		500		{object}	Response
//	@Router			/v1/orders [post]
func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	const maxCheckoutItems = 50

	var req checkoutRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if len(req.Items) > maxCheckoutItems {
		ErrorResponse(w, http.StatusBadRequest, "TOO_MANY_ITEMS", fmt.Sprintf("checkout is limited to %d items per order", maxCheckoutItems))
		return
	}

	userID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	idempotencyKey, err := uuid.Parse(req.IdempotencyKey)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "idempotency_key must be a valid UUID")
		return
	}

	if len(req.Items) == 0 {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ITEMS", "items must not be empty")
		return
	}

	items := make([]service.CheckoutItem, len(req.Items))
	for i, it := range req.Items {
		skuID, err := uuid.Parse(it.SKUID)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_SKU_ID", fmt.Sprintf("items[%d]: sku_id must be a valid UUID", i))
			return
		}
		if it.Quantity <= 0 {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_QUANTITY", fmt.Sprintf("items[%d]: quantity must be greater than zero", i))
			return
		}
		if it.UnitPrice <= 0 {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_UNIT_PRICE", fmt.Sprintf("items[%d]: unit_price must be greater than zero", i))
			return
		}
		item := service.CheckoutItem{
			SKUID:     skuID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
		if it.SellerID != "" {
			sid, err := uuid.Parse(it.SellerID)
			if err != nil {
				ErrorResponse(w, http.StatusBadRequest, "INVALID_SELLER_ID", fmt.Sprintf("items[%d]: seller_id must be a valid UUID", i))
				return
			}
			item.SellerID = &sid
		}
		items[i] = item
	}

	order, err := h.svc.Checkout(r.Context(), userID, idempotencyKey, items, req.Currency)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, order)
}

// GetOrder handles GET /v1/orders/{id}.
//
//	@Summary		Get an order
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{object}	Response{data=object}
//	@Failure		400	{object}	Response
//	@Failure		401	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/v1/orders/{id} [get]
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "order id must be a valid UUID")
		return
	}

	userID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	order, items, err := h.svc.GetOrder(r.Context(), orderID, userID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, orderResponse{Order: order, Items: items})
}

// ListOrders handles GET /v1/orders.
//
//	@Summary		List orders for the authenticated user
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Page size (default 20)"
//	@Param			offset	query		int	false	"Page offset (default 0)"
//	@Success		200		{object}	Response{data=[]domain.Order}
//	@Failure		401		{object}	Response
//	@Router			/v1/orders [get]
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	userID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	limit, offset := parsePage(r)
	orders, total, err := h.svc.ListOrders(r.Context(), userID, limit, offset)
	if err != nil {
		HandleError(w, err)
		return
	}

	page := offset/limit + 1
	httpx.Paginated(w, http.StatusOK, orders, total, page, limit)
}

// ListSellerOrders handles GET /v1/orders/seller.
//
//	@Summary		List orders for the authenticated seller
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Page size (default 20)"
//	@Param			offset	query		int	false	"Page offset (default 0)"
//	@Success		200		{object}	Response{data=[]domain.Order}
//	@Failure		401		{object}	Response
//	@Router			/v1/orders/seller [get]
func (h *OrderHandler) ListSellerOrders(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	sellerID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	limit, offset := parsePage(r)
	orders, total, err := h.svc.ListSellerOrders(r.Context(), sellerID, limit, offset)
	if err != nil {
		HandleError(w, err)
		return
	}

	page := offset/limit + 1
	httpx.Paginated(w, http.StatusOK, orders, total, page, limit)
}

// parsePage reads limit and offset from query params, with safe defaults.
// limit is clamped to [1, 100] to prevent DoS via oversized queries.
func parsePage(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

// GetSellerOrder handles GET /v1/orders/seller/{id}.
//
//	@Summary		Get a single order for the authenticated seller
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{object}	Response{data=object}
//	@Failure		401	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/v1/orders/seller/{id} [get]
func (h *OrderHandler) GetSellerOrder(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "order id must be a valid UUID")
		return
	}

	sellerID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	order, items, err := h.svc.GetSellerOrder(r.Context(), orderID, sellerID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, orderResponse{Order: order, Items: items})
}

// CancelOrder handles POST /v1/orders/{id}/cancel.
//
//	@Summary		Cancel an order
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{object}	Response{data=domain.Order}
//	@Failure		400	{object}	Response
//	@Failure		401	{object}	Response
//	@Failure		404	{object}	Response
//	@Failure		409	{object}	Response
//	@Router			/v1/orders/{id}/cancel [post]
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "order id must be a valid UUID")
		return
	}

	userID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	order, err := h.svc.CancelOrder(r.Context(), orderID, userID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, order)
}
