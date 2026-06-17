package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	gw "github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/registry"
)

// Handler exposes the gateway admin API.
// All endpoints require role=admin in the JWT (enforced by the caller via authMW.RequireRole).
type Handler struct {
	db  *sql.DB
	reg *registry.RedisRegistry
}

func NewHandler(db *sql.DB, reg *registry.RedisRegistry) *Handler {
	return &Handler{db: db, reg: reg}
}

// Mount registers admin routes onto r. authMW.Authenticate + RequireRole("admin")
// must already be applied to the sub-router.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/routes", h.listRoutes)
	r.Post("/routes", h.createRoute)
	r.Put("/routes/{id}", h.updateRoute)
	r.Delete("/routes/{id}", h.deleteRoute)
	r.Get("/audit", h.queryAudit)
	r.Get("/registry", h.listRegistry)
}

type routeRow struct {
	ID          string    `json:"id"`
	PathPrefix  string    `json:"path_prefix"`
	Upstream    string    `json:"upstream"`
	AuthMode    string    `json:"auth_mode"`
	StripPrefix bool      `json:"strip_prefix"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *Handler) listRoutes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, path_prefix, upstream, auth_mode, strip_prefix, enabled, created_at, updated_at
		 FROM gateway_routes ORDER BY path_prefix`)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var result []routeRow
	for rows.Next() {
		var row routeRow
		if err := rows.Scan(&row.ID, &row.PathPrefix, &row.Upstream, &row.AuthMode,
			&row.StripPrefix, &row.Enabled, &row.CreatedAt, &row.UpdatedAt); err != nil {
			jsonErr(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		result = append(result, row)
	}
	jsonOK(w, map[string]any{"routes": result})
}

type createRouteReq struct {
	PathPrefix  string `json:"path_prefix"`
	Upstream    string `json:"upstream"`
	AuthMode    string `json:"auth_mode"`
	StripPrefix bool   `json:"strip_prefix"`
}

func (h *Handler) createRoute(w http.ResponseWriter, r *http.Request) {
	var req createRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if req.PathPrefix == "" || req.Upstream == "" {
		jsonErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "path_prefix and upstream are required")
		return
	}
	if req.AuthMode == "" {
		req.AuthMode = "required"
	}

	var id string
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		req.PathPrefix, req.Upstream, req.AuthMode, req.StripPrefix,
	).Scan(&id)
	if err != nil {
		jsonErr(w, http.StatusConflict, "CREATE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"id": id})
}

type updateRouteReq struct {
	Upstream    *string `json:"upstream"`
	AuthMode    *string `json:"auth_mode"`
	StripPrefix *bool   `json:"strip_prefix"`
	Enabled     *bool   `json:"enabled"`
}

func (h *Handler) updateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Build dynamic SET clause.
	sets := []string{}
	args := []any{}
	idx := 1
	if req.Upstream != nil {
		sets = append(sets, "upstream = $"+strconv.Itoa(idx))
		args = append(args, *req.Upstream)
		idx++
	}
	if req.AuthMode != nil {
		sets = append(sets, "auth_mode = $"+strconv.Itoa(idx))
		args = append(args, *req.AuthMode)
		idx++
	}
	if req.StripPrefix != nil {
		sets = append(sets, "strip_prefix = $"+strconv.Itoa(idx))
		args = append(args, *req.StripPrefix)
		idx++
	}
	if req.Enabled != nil {
		sets = append(sets, "enabled = $"+strconv.Itoa(idx))
		args = append(args, *req.Enabled)
		idx++
	}
	if len(sets) == 0 {
		jsonErr(w, http.StatusBadRequest, "NOTHING_TO_UPDATE", "provide at least one field")
		return
	}

	q := "UPDATE gateway_routes SET "
	for i, s := range sets {
		if i > 0 {
			q += ", "
		}
		q += s
	}
	q += " WHERE id = $" + strconv.Itoa(idx)
	args = append(args, id)

	res, err := h.db.ExecContext(r.Context(), q, args...)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	jsonOK(w, map[string]bool{"updated": true})
}

func (h *Handler) deleteRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE gateway_routes SET enabled = false WHERE id = $1`, id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		jsonErr(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	jsonOK(w, map[string]bool{"disabled": true})
}

type auditRow struct {
	ID         int64     `json:"id"`
	Ts         time.Time `json:"ts"`
	RequestID  string    `json:"request_id"`
	UserID     *string   `json:"user_id,omitempty"`
	IP         string    `json:"ip"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Upstream   string    `json:"upstream"`
	StatusCode int       `json:"status_code"`
	Event      string    `json:"event"`
	Detail     string    `json:"detail"`
}

func (h *Handler) queryAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 100
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	args := []any{}
	where := "WHERE 1=1"
	idx := 1
	if userID := q.Get("user_id"); userID != "" {
		where += " AND user_id = $" + strconv.Itoa(idx)
		args = append(args, userID)
		idx++
	}
	if event := q.Get("event"); event != "" {
		where += " AND event = $" + strconv.Itoa(idx)
		args = append(args, event)
		idx++
	}
	if from := q.Get("from"); from != "" {
		where += " AND ts >= $" + strconv.Itoa(idx)
		args = append(args, from)
		idx++
	}
	if to := q.Get("to"); to != "" {
		where += " AND ts <= $" + strconv.Itoa(idx)
		args = append(args, to)
		idx++
	}
	args = append(args, limit)

	sql := `SELECT id, ts, request_id, user_id, ip, method, path, upstream, status_code, event, detail
			FROM gateway_audit_log ` + where + ` ORDER BY ts DESC LIMIT $` + strconv.Itoa(idx)

	rows, err := h.db.QueryContext(r.Context(), sql, args...)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var result []auditRow
	for rows.Next() {
		var row auditRow
		if err := rows.Scan(&row.ID, &row.Ts, &row.RequestID, &row.UserID, &row.IP,
			&row.Method, &row.Path, &row.Upstream, &row.StatusCode, &row.Event, &row.Detail); err != nil {
			jsonErr(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		result = append(result, row)
	}
	jsonOK(w, map[string]any{"entries": result, "count": len(result)})
}

type instanceEntry struct {
	Service    string    `json:"service"`
	InstanceID string    `json:"instance_id"`
	Addr       string    `json:"addr"`
	StartedAt  time.Time `json:"started_at"`
}

func (h *Handler) listRegistry(w http.ResponseWriter, r *http.Request) {
	services, err := h.reg.AllInstances(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "REGISTRY_ERROR", err.Error())
		return
	}
	var entries []instanceEntry
	for svc, insts := range services {
		for _, inst := range insts {
			entries = append(entries, instanceEntry{
				Service:    svc,
				InstanceID: inst.InstanceID,
				Addr:       inst.Addr,
				StartedAt:  inst.StartedAt,
			})
		}
	}
	jsonOK(w, map[string]any{"instances": entries, "count": len(entries)})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": v})
}

func jsonErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   map[string]string{"code": code, "message": msg},
	})
}

// RequireAdmin returns a chi middleware that enforces role=admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := gw.UserFromContext(r.Context())
		if u == nil || u.Role != "admin" {
			jsonErr(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
