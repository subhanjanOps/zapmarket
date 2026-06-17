package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/authctx"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/service"
)

type OrderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type checkoutItemRequest struct {
	SKUID     string `json:"sku_id"`
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

	var req checkoutRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
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

	items := make([]service.CheckoutItem, len(req.Items))
	for i, it := range req.Items {
		skuID, err := uuid.Parse(it.SKUID)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_SKU_ID", "items["+string(rune('0'+i))+"]: sku_id must be a valid UUID")
			return
		}
		items[i] = service.CheckoutItem{
			SKUID:     skuID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
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

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"order": order,
		"items": items,
	})
}

// ListOrders handles GET /v1/orders.
//
//	@Summary		List orders for the authenticated user
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	Response{data=[]domain.Order}
//	@Failure		401	{object}	Response
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

	orders, err := h.svc.ListOrders(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, orders)
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
