# HTTP Metrics Middleware Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a shared `MetricsMiddleware` to `pkg/metrics` that automatically increments `<namespace>_http_requests_total` for every HTTP request, then wire it into all six services so the Grafana "HTTP Traffic" panel shows real data.

**Architecture:** A `responseWriter` wrapper captures the status code written by the downstream handler. `Middleware(m *Base) func(http.Handler) http.Handler` wraps any `http.Handler` and increments `RequestsTotal` with `method` and a 3-digit status string after the handler returns. Each service wraps its top-level mux/router with this middleware in `main.go`.

**Tech Stack:** Go stdlib `net/http`, `github.com/prometheus/client_golang`, `github.com/go-chi/chi/v5` (product-catalog, order-management services).

## Global Constraints

- No new dependencies — use only packages already in `go.work`
- `status` label value must be the 3-digit HTTP status code as a string (e.g. `"200"`, `"404"`) — matches the existing Grafana dashboard query that groups by `status`
- `method` label value is `r.Method` (uppercase, e.g. `"GET"`, `"POST"`)
- Do not touch `currency-service` — it has its own `infrastructure/metrics` package separate from `pkg/metrics`
- Do not touch `api-gateway` — it already has its own `metrics.NewTracker()` mechanism
- Do not touch `notification-service` — it has no HTTP routes (only health + metrics endpoints, no business traffic)
- Services to wire: `auth-service`, `product-catalog-service`, `inventory-service`, `order-management-service`, `payment-service`

---

## File Map

| Action | File | Change |
|--------|------|--------|
| Modify | `pkg/metrics/metrics.go` | Add `responseWriter` struct + `Middleware` function |
| Create | `pkg/metrics/metrics_test.go` | Unit tests for `Middleware` |
| Modify | `services/auth-service/cmd/main.go` | Wrap mux with `m.Middleware()` |
| Modify | `services/product-catalog-service/main.go` | Add `r.Use(m.Middleware())` on chi router |
| Modify | `services/inventory-service/main.go` | Wrap mux with `m.Middleware()` |
| Modify | `services/order-management-service/main.go` | Add `r.Use(m.Middleware())` on chi router |
| Modify | `services/payment-service/main.go` | Wrap mux with `m.Middleware()` |

---

## Task 1: Add `Middleware` to `pkg/metrics`

**Files:**
- Modify: `pkg/metrics/metrics.go`
- Create: `pkg/metrics/metrics_test.go`

**Interfaces:**
- Produces: `func (m *Base) Middleware() func(http.Handler) http.Handler`
  - The returned middleware is chi-compatible (satisfies `func(http.Handler) http.Handler`)
  - Also usable with stdlib mux via `m.Middleware()(mux)`

- [ ] **Step 1: Write the failing test**

Create `pkg/metrics/metrics_test.go`:

```go
package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/zapmarket/zapmarket/pkg/metrics"
)

func TestMiddleware_IncrementsCounter(t *testing.T) {
	m := metrics.New("testsvc")

	handler := m.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/foo", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	count := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("POST", "201"))
	if count != 1 {
		t.Fatalf("expected counter=1, got %v", count)
	}
}

func TestMiddleware_DefaultsTo200(t *testing.T) {
	m := metrics.New("testsvc2")

	handler := m.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no explicit WriteHeader — defaults to 200
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/bar", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	count := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "200"))
	if count != 1 {
		t.Fatalf("expected counter=1, got %v", count)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd pkg/metrics
go test ./... -v -run TestMiddleware
```

Expected: FAIL — `m.Middleware undefined`

- [ ] **Step 3: Implement `Middleware` in `pkg/metrics/metrics.go`**

Add after the existing `Handler()` method:

```go
// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware returns an http.Handler middleware that records
// request counts by method and HTTP status code.
func (m *Base) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)
			status := rw.status
			if status == 0 {
				status = http.StatusOK
			}
			m.RequestsTotal.WithLabelValues(r.Method, fmt.Sprintf("%d", status)).Inc()
		})
	}
}
```

Also add `"fmt"` to the import block in `metrics.go`.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd pkg/metrics
go test ./... -v -run TestMiddleware
```

Expected: PASS — both `TestMiddleware_IncrementsCounter` and `TestMiddleware_DefaultsTo200`

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/metrics.go pkg/metrics/metrics_test.go
git commit -m "feat(metrics): add HTTP Middleware that increments RequestsTotal per request"
```

---

## Task 2: Wire middleware into `auth-service`

**Files:**
- Modify: `services/auth-service/cmd/main.go`

**Interfaces:**
- Consumes: `m.Middleware()` from Task 1 — `func(http.Handler) http.Handler`

`auth-service` uses stdlib `http.NewServeMux`. The middleware wraps the whole mux so every registered route is instrumented, including health and swagger — which is fine and consistent.

- [ ] **Step 1: Wrap the mux in `main.go`**

In `services/auth-service/cmd/main.go`, locate the `http.Server` construction (around line 190). It currently reads:

```go
srv := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler: mux,
    ...
}
```

Change `Handler: mux` to `Handler: m.Middleware()(mux)`:

```go
srv := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler: m.Middleware()(mux),
    ...
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd services/auth-service
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add services/auth-service/cmd/main.go
git commit -m "feat(auth-service): instrument HTTP requests via pkg/metrics Middleware"
```

---

## Task 3: Wire middleware into `product-catalog-service`

**Files:**
- Modify: `services/product-catalog-service/main.go`

**Interfaces:**
- Consumes: `m.Middleware()` from Task 1 — chi-compatible `func(http.Handler) http.Handler`

`product-catalog-service` uses chi. Add the middleware via `r.Use()` so it applies to all routes. Place it before other middleware so it captures the final status.

- [ ] **Step 1: Add `r.Use(m.Middleware())` to the chi router**

In `services/product-catalog-service/main.go`, locate the router setup (around line 139):

```go
r := chi.NewRouter()
r.Use(chimiddleware.Logger)
r.Use(chimiddleware.Recoverer)
r.Use(chimiddleware.RequestID)
r.Use(httpx.LimitBody(httpx.MaxBodyBytes))
```

Add `m.Middleware()` as the **first** middleware:

```go
r := chi.NewRouter()
r.Use(m.Middleware())
r.Use(chimiddleware.Logger)
r.Use(chimiddleware.Recoverer)
r.Use(chimiddleware.RequestID)
r.Use(httpx.LimitBody(httpx.MaxBodyBytes))
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd services/product-catalog-service
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add services/product-catalog-service/main.go
git commit -m "feat(product-catalog-service): instrument HTTP requests via pkg/metrics Middleware"
```

---

## Task 4: Wire middleware into `inventory-service`

**Files:**
- Modify: `services/inventory-service/main.go`

**Interfaces:**
- Consumes: `m.Middleware()` from Task 1 — `func(http.Handler) http.Handler`

`inventory-service` uses stdlib `http.NewServeMux`.

- [ ] **Step 1: Locate the `http.Server` construction in `main.go`**

Find the server instantiation. It will look similar to:

```go
srv := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler: mux,
    ...
}
```

Change `Handler: mux` to `Handler: m.Middleware()(mux)`:

```go
srv := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler: m.Middleware()(mux),
    ...
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd services/inventory-service
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add services/inventory-service/main.go
git commit -m "feat(inventory-service): instrument HTTP requests via pkg/metrics Middleware"
```

---

## Task 5: Wire middleware into `order-management-service`

**Files:**
- Modify: `services/order-management-service/main.go`

**Interfaces:**
- Consumes: `m.Middleware()` from Task 1 — chi-compatible `func(http.Handler) http.Handler`

`order-management-service` uses chi.

- [ ] **Step 1: Add `r.Use(m.Middleware())` to the chi router**

In `services/order-management-service/main.go`, locate the chi router setup (around line 130). Add `m.Middleware()` as the first `r.Use()` call:

```go
r := chi.NewRouter()
r.Use(m.Middleware())
// ... existing middleware ...
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd services/order-management-service
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add services/order-management-service/main.go
git commit -m "feat(order-management-service): instrument HTTP requests via pkg/metrics Middleware"
```

---

## Task 6: Wire middleware into `payment-service`

**Files:**
- Modify: `services/payment-service/main.go`

**Interfaces:**
- Consumes: `m.Middleware()` from Task 1 — `func(http.Handler) http.Handler`

`payment-service` uses stdlib `http.NewServeMux`.

- [ ] **Step 1: Locate the `http.Server` construction in `main.go`**

Find the server instantiation and change `Handler: mux` to `Handler: m.Middleware()(mux)`:

```go
srv := &http.Server{
    Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler: m.Middleware()(mux),
    ...
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd services/payment-service
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add services/payment-service/main.go
git commit -m "feat(payment-service): instrument HTTP requests via pkg/metrics Middleware"
```

---

## Task 7: End-to-end smoke test

- [ ] **Step 1: Rebuild and restart all services**

```bash
docker compose build auth-service product-catalog-service inventory-service order-management-service payment-service
docker compose up -d auth-service product-catalog-service inventory-service order-management-service payment-service
```

- [ ] **Step 2: Send test traffic**

```bash
# auth-service
curl -s http://localhost:8080/health

# product-catalog-service
curl -s http://localhost:8081/v1/products

# inventory-service
curl -s http://localhost:8082/health

# order-management-service
curl -s http://localhost:8084/health

# payment-service
curl -s http://localhost:8083/health
```

- [ ] **Step 3: Verify metrics are being scraped**

```bash
curl -s http://localhost:8080/metrics | grep http_requests_total
curl -s http://localhost:8081/metrics | grep http_requests_total
```

Expected output (example for auth):
```
auth_http_requests_total{method="GET",status="200"} 1
```

- [ ] **Step 4: Check Grafana dashboard**

Open `http://localhost:3009/d/zapmarket-overview`.  
The **HTTP Traffic** panel should now display request rate lines per service.
