package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/service"
)

type AdminOrderHandler struct {
	svc service.OrderService
	db  *sql.DB
}

func NewAdminOrderHandler(svc service.OrderService) *AdminOrderHandler {
	return &AdminOrderHandler{svc: svc}
}

func NewAdminOrderHandlerWithDB(svc service.OrderService, db *sql.DB) *AdminOrderHandler {
	return &AdminOrderHandler{svc: svc, db: db}
}

// AdminListOrders handles GET /v1/admin/orders
func (h *AdminOrderHandler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
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

	if orders == nil {
		orders = []*domain.Order{}
	}
	page := 1
	if limit > 0 && offset > 0 {
		page = offset/limit + 1
	}
	httpx.Paginated(w, http.StatusOK, orders, total, page, limit)
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

// GetGMV handles GET /v1/admin/analytics/gmv?days=N
func (h *AdminOrderHandler) GetGMV(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		ErrorResponse(w, http.StatusServiceUnavailable, "NO_DB", "analytics not configured")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DATE(created_at)::TEXT AS date,
		       COALESCE(SUM(total_amount), 0) AS amount_paise
		FROM orders
		WHERE status = 'CONFIRMED'
		  AND created_at >= NOW() - ($1 || ' days')::INTERVAL
		GROUP BY DATE(created_at)
		ORDER BY date
	`, days)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	defer rows.Close()

	type dailyEntry struct {
		Date        string `json:"date"`
		AmountPaise int64  `json:"amount_paise"`
	}
	var daily []dailyEntry
	for rows.Next() {
		var e dailyEntry
		if err := rows.Scan(&e.Date, &e.AmountPaise); err != nil {
			continue
		}
		daily = append(daily, e)
	}
	if daily == nil {
		daily = []dailyEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"daily": daily}) //nolint:errcheck
}

// GetFunnel handles GET /v1/admin/analytics/funnel?days=N
func (h *AdminOrderHandler) GetFunnel(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		ErrorResponse(w, http.StatusServiceUnavailable, "NO_DB", "analytics not configured")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT status, COUNT(*) AS count
		FROM orders
		WHERE created_at >= NOW() - ($1 || ' days')::INTERVAL
		GROUP BY status
		ORDER BY count DESC
	`, days)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	defer rows.Close()

	type stage struct {
		Stage string `json:"stage"`
		Count int64  `json:"count"`
	}
	var stages []stage
	for rows.Next() {
		var s stage
		if err := rows.Scan(&s.Stage, &s.Count); err != nil {
			continue
		}
		stages = append(stages, s)
	}
	if stages == nil {
		stages = []stage{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"stages": stages}) //nolint:errcheck
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
