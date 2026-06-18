package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/service"
)

type AdminOrderHandler struct {
	svc service.OrderService
}

func NewAdminOrderHandler(svc service.OrderService) *AdminOrderHandler {
	return &AdminOrderHandler{svc: svc}
}

// AdminListOrders handles GET /v1/admin/orders
func (h *AdminOrderHandler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	params := contracts.OrderListParams{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	}

	if uid := q.Get("user_id"); uid != "" {
		id, err := uuid.Parse(uid)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a valid UUID")
			return
		}
		params.UserID = &id
	}
	if from := q.Get("from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_FROM", "from must be YYYY-MM-DD")
			return
		}
		params.From = &t
	}
	if to := q.Get("to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_TO", "to must be YYYY-MM-DD")
			return
		}
		// include the entire day
		end := t.Add(24*time.Hour - time.Second)
		params.To = &end
	}

	orders, total, err := h.svc.ListAllOrders(r.Context(), params)
	if err != nil {
		HandleError(w, err)
		return
	}

	page := 1
	if limit > 0 && offset > 0 {
		page = offset/limit + 1
	}

	type listResp struct {
		Data     interface{} `json:"data"`
		Total    int64       `json:"total"`
		Page     int         `json:"page"`
		PageSize int         `json:"page_size"`
		Success  bool        `json:"success"`
	}
	if orders == nil {
		orders = []*domain.Order{}
	}
	JSON(w, http.StatusOK, listResp{
		Success:  true,
		Data:     orders,
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}

// AdminGetOrder handles GET /v1/admin/orders/{id}
func (h *AdminOrderHandler) AdminGetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "order id must be a valid UUID")
		return
	}
	order, items, err := h.svc.AdminGetOrder(r.Context(), orderID)
	if err != nil {
		HandleError(w, err)
		return
	}
	SuccessResponse(w, http.StatusOK, orderResponse{Order: order, Items: items})
}

// AdminCancelOrder handles POST /v1/admin/orders/{id}/cancel
func (h *AdminOrderHandler) AdminCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "order id must be a valid UUID")
		return
	}
	order, err := h.svc.AdminCancelOrder(r.Context(), orderID)
	if err != nil {
		HandleError(w, err)
		return
	}
	SuccessResponse(w, http.StatusOK, order)
}
