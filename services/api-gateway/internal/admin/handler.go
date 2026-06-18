package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	gw "github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/metrics"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/registry"

	goredis "github.com/redis/go-redis/v9"
)

// Handler exposes the gateway admin API.
type Handler struct {
	db      *sql.DB
	reg     *registry.RedisRegistry
	tracker *metrics.Tracker
	rdb     *goredis.Client
	resolve func(ctx context.Context, name string) (string, bool)
}

func NewHandler(
	db *sql.DB,
	reg *registry.RedisRegistry,
	tracker *metrics.Tracker,
	rdb *goredis.Client,
	resolve func(ctx context.Context, name string) (string, bool),
) *Handler {
	return &Handler{db: db, reg: reg, tracker: tracker, rdb: rdb, resolve: resolve}
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/routes", h.listRoutes)
	r.Post("/routes", h.createRoute)
	r.Put("/routes/{id}", h.updateRoute)
	r.Delete("/routes/{id}", h.deleteRoute)

	r.Get("/audit", h.queryAudit)
	r.Get("/registry", h.listRegistry)
	r.Get("/metrics", h.getMetrics)
	r.Get("/stats", h.getStats)

	r.Post("/probe", h.probeRoute)

	r.Get("/blocklist", h.listBlocklist)
	r.Post("/blocklist", h.addToBlocklist)
	r.Delete("/blocklist/{ip}", h.removeFromBlocklist)
}

// ── Routes ───────────────────────────────────────────────────────────────────

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
		 FROM gateway_routes ORDER BY length(path_prefix) DESC, path_prefix`)
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

	var sb strings.Builder
	sb.WriteString("UPDATE gateway_routes SET ")
	for i, s := range sets {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(s)
	}
	sb.WriteString(" WHERE id = $")
	sb.WriteString(strconv.Itoa(idx))
	q := sb.String()
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

// ── Audit ─────────────────────────────────────────────────────────────────────

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
	// after_id supports real-time tail: only return entries newer than this ID
	if afterID := q.Get("after_id"); afterID != "" {
		where += " AND id > $" + strconv.Itoa(idx)
		args = append(args, afterID)
		idx++
	}
	args = append(args, limit)

	query := `SELECT id, ts, request_id, user_id, ip, method, path, upstream, status_code, event, detail
			  FROM gateway_audit_log ` + where + ` ORDER BY ts DESC LIMIT $` + strconv.Itoa(idx)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
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

// ── Registry with health checks ───────────────────────────────────────────────

type instanceEntry struct {
	Service    string    `json:"service"`
	InstanceID string    `json:"instance_id"`
	Addr       string    `json:"addr"`
	StartedAt  time.Time `json:"started_at"`
	Healthy    bool      `json:"healthy"`
}

func (h *Handler) listRegistry(w http.ResponseWriter, r *http.Request) {
	services, err := h.reg.AllInstances(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "REGISTRY_ERROR", err.Error())
		return
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		entries []instanceEntry
	)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}

	for svc, insts := range services {
		for _, inst := range insts {
			entry := instanceEntry{
				Service:    svc,
				InstanceID: inst.InstanceID,
				Addr:       inst.Addr,
				StartedAt:  inst.StartedAt,
			}
			wg.Add(1)
			go func(e instanceEntry) {
				defer wg.Done()
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, e.Addr+"/health", nil)
				resp, err := client.Do(req)
				if err == nil {
					_ = resp.Body.Close()
					e.Healthy = resp.StatusCode < 500
				}
				mu.Lock()
				entries = append(entries, e)
				mu.Unlock()
			}(entry)
		}
	}

	wg.Wait()
	jsonOK(w, map[string]any{"instances": entries, "count": len(entries)})
}

// ── Metrics ───────────────────────────────────────────────────────────────────

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	snapshots := h.tracker.Snapshots()
	jsonOK(w, map[string]any{"upstreams": snapshots})
}

// ── Stats (overview dashboard) ────────────────────────────────────────────────

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Active / total routes
	var activeRoutes, totalRoutes int
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gateway_routes`).Scan(&totalRoutes)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gateway_routes WHERE enabled = true`).Scan(&activeRoutes)

	// Live instances
	services, _ := h.reg.AllInstances(ctx)
	liveInstances := 0
	for _, insts := range services {
		liveInstances += len(insts)
	}

	// Recent audit entries (last 5)
	rows, err := h.db.QueryContext(ctx,
		`SELECT id, ts, request_id, user_id, ip, method, path, upstream, status_code, event, detail
		 FROM gateway_audit_log ORDER BY ts DESC LIMIT 5`)
	var recentAudit []auditRow
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var row auditRow
			if err := rows.Scan(&row.ID, &row.Ts, &row.RequestID, &row.UserID, &row.IP,
				&row.Method, &row.Path, &row.Upstream, &row.StatusCode, &row.Event, &row.Detail); err == nil {
				recentAudit = append(recentAudit, row)
			}
		}
	}

	// Upstream stats summary
	snapshots := h.tracker.Snapshots()

	jsonOK(w, map[string]any{
		"active_routes":  activeRoutes,
		"total_routes":   totalRoutes,
		"live_instances": liveInstances,
		"live_services":  len(services),
		"recent_audit":   recentAudit,
		"upstreams":      snapshots,
	})
}

// ── Probe ─────────────────────────────────────────────────────────────────────

type probeReq struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Token   string            `json:"token,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type probeResp struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	LatencyMs int64             `json:"latency_ms"`
	Upstream  string            `json:"upstream"`
	Addr      string            `json:"addr"`
}

func (h *Handler) probeRoute(w http.ResponseWriter, r *http.Request) {
	var req probeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	if req.Path == "" {
		jsonErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "path is required")
		return
	}

	// Find the best matching route from DB.
	var upstreamName string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT upstream FROM gateway_routes
		 WHERE enabled = true AND ($1 = path_prefix OR $1 LIKE path_prefix || '/%')
		 ORDER BY length(path_prefix) DESC LIMIT 1`,
		req.Path,
	).Scan(&upstreamName)
	if err == sql.ErrNoRows {
		jsonErr(w, http.StatusNotFound, "NO_ROUTE", fmt.Sprintf("no enabled route matches %s", req.Path))
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	addr, ok := h.resolve(r.Context(), upstreamName)
	if !ok {
		jsonErr(w, http.StatusBadGateway, "NO_UPSTREAM", fmt.Sprintf("no healthy instance for %s", upstreamName))
		return
	}

	targetURL := addr + req.Path

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	outReq, err := http.NewRequestWithContext(r.Context(), req.Method, targetURL, bodyReader)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "REQUEST_BUILD_FAILED", err.Error())
		return
	}
	if req.Token != "" {
		outReq.Header.Set("Authorization", "Bearer "+req.Token)
	}
	for k, v := range req.Headers {
		outReq.Header.Set(k, v)
	}
	if req.Body != "" {
		outReq.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	resp, err := client.Do(outReq)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		jsonErr(w, http.StatusBadGateway, "PROBE_FAILED", err.Error())
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024)) // cap at 32KB
	respHeaders := make(map[string]string)
	for k, vs := range resp.Header {
		respHeaders[k] = strings.Join(vs, ", ")
	}

	jsonOK(w, probeResp{
		Status:    resp.StatusCode,
		Headers:   respHeaders,
		Body:      string(bodyBytes),
		LatencyMs: latency,
		Upstream:  upstreamName,
		Addr:      addr,
	})
}

// ── Blocklist ─────────────────────────────────────────────────────────────────

type blocklistEntry struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	BlockedAt time.Time `json:"blocked_at"`
}

func (h *Handler) listBlocklist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ips, err := h.rdb.SMembers(ctx, gw.BlocklistKey).Result()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "REDIS_ERROR", err.Error())
		return
	}

	entries := make([]blocklistEntry, 0, len(ips))
	for _, ip := range ips {
		meta, _ := h.rdb.HGet(ctx, gw.BlocklistMetaKey, ip).Result()
		entry := blocklistEntry{IP: ip, BlockedAt: time.Now()}
		if meta != "" {
			_ = json.Unmarshal([]byte(meta), &entry)
			entry.IP = ip
		}
		entries = append(entries, entry)
	}
	jsonOK(w, map[string]any{"blocked": entries, "count": len(entries)})
}

type addBlockReq struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

func (h *Handler) addToBlocklist(w http.ResponseWriter, r *http.Request) {
	var req addBlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	req.IP = strings.TrimSpace(req.IP)
	if req.IP == "" {
		jsonErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "ip is required")
		return
	}
	if net.ParseIP(req.IP) == nil {
		jsonErr(w, http.StatusBadRequest, "INVALID_IP", "not a valid IP address")
		return
	}

	entry := blocklistEntry{IP: req.IP, Reason: req.Reason, BlockedAt: time.Now()}
	meta, _ := json.Marshal(entry)

	ctx := r.Context()
	if err := h.rdb.SAdd(ctx, gw.BlocklistKey, req.IP).Err(); err != nil {
		jsonErr(w, http.StatusInternalServerError, "REDIS_ERROR", err.Error())
		return
	}
	_ = h.rdb.HSet(ctx, gw.BlocklistMetaKey, req.IP, string(meta)).Err()

	jsonOK(w, map[string]bool{"blocked": true})
}

func (h *Handler) removeFromBlocklist(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")
	ctx := r.Context()
	if err := h.rdb.SRem(ctx, gw.BlocklistKey, ip).Err(); err != nil {
		jsonErr(w, http.StatusInternalServerError, "REDIS_ERROR", err.Error())
		return
	}
	_ = h.rdb.HDel(ctx, gw.BlocklistMetaKey, ip).Err()
	jsonOK(w, map[string]bool{"removed": true})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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

// RequireAdmin enforces role=admin on the sub-router.
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
