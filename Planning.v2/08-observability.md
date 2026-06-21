# Phase 8 — Observability

**Goal:** Production-grade visibility into every service: distributed traces, structured metrics, and dashboards. Can be started in parallel with any other phase.

---

## 8.1 Current State

- Each service uses `log/slog` with structured JSON output — good baseline.
- api-gateway has a `metrics.Tracker` for per-upstream latency/count.
- No distributed tracing (no trace/span IDs propagated across services).
- No metrics export (Prometheus, Datadog, etc.).
- No alerting.

---

## 8.2 OpenTelemetry Tracing

### Instrument each service

Add `go.opentelemetry.io/otel` + the OTLP exporter to each service's `go.mod`:

```go
// In cmd/main.go of each service:
tp := oteltracing.NewTracerProvider(
    oteltracing.WithBatcher(otlptracehttp.NewExporter(...)),
    oteltracing.WithResource(resource.NewWithAttributes(
        semconv.ServiceName("<service-name>"),
        semconv.ServiceVersion("1.0.0"),
    )),
)
otel.SetTracerProvider(tp)
otel.SetTextMapPropagator(propagation.TraceContext{})
defer tp.Shutdown(ctx)
```

### Auto-instrumentation

- HTTP: wrap `http.Handler` with `otelhttp.NewHandler(handler, "service-name")`
- DB: wrap `*sql.DB` with `otelsql` (`github.com/XSAM/otelsql`)
- Redis: use `go.opentelemetry.io/contrib/instrumentation/github.com/redis/go-redis/otelredis`
- gRPC: use `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
- Kafka: manually add trace context to message headers on produce; extract on consume

### Trace propagation across services

The api-gateway already forwards `X-Request-ID`. Extend it to forward `traceparent` and `tracestate` W3C headers to all upstream services. Each service extracts these and continues the trace span.

### Collector

Add Jaeger (all-in-one) to `docker-compose.yml`:

```yaml
jaeger:
  image: jaegertracing/all-in-one:1.55
  container_name: zapmarket-jaeger
  ports:
    - "16686:16686"   # UI
    - "4318:4318"     # OTLP HTTP receiver
  networks:
    - zapnet
```

Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://zapmarket-jaeger:4318` in each service.

---

## 8.3 Prometheus Metrics

Each service exposes `/metrics` in Prometheus format via `github.com/prometheus/client_golang`:

### Standard metrics (auto-instrumented)

- `http_request_duration_seconds` (histogram by route, method, status)
- `http_requests_total` (counter by route, method, status)
- `grpc_server_handling_seconds` (histogram by method, status)
- `db_query_duration_seconds` (histogram by query label)
- `redis_command_duration_seconds` (histogram by command)

### Custom business metrics

| Metric | Type | Service |
|--------|------|---------|
| `orders_created_total` | counter | order-management |
| `orders_confirmed_total` | counter | order-management |
| `payments_captured_total` | counter | payment |
| `payments_failed_total` | counter | payment |
| `inventory_reservations_total` | counter | inventory |
| `inventory_insufficient_stock_total` | counter | inventory |
| `circuit_breaker_state` | gauge (0=closed, 1=open) | api-gateway |
| `kafka_consumer_lag` | gauge | notification, order |
| `outbox_pending_rows` | gauge | all services with outbox |

### Prometheus + Grafana in docker-compose

```yaml
prometheus:
  image: prom/prometheus:v2.50.0
  volumes:
    - ./infra/prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - "9090:9090"
  networks:
    - zapnet

grafana:
  image: grafana/grafana:10.3.0
  ports:
    - "3000:3000"
  volumes:
    - grafana-data:/var/lib/grafana
    - ./infra/grafana/dashboards:/etc/grafana/provisioning/dashboards
  networks:
    - zapnet
```

`infra/prometheus.yml` scrapes `/metrics` from all services.

---

## 8.4 Dashboards

### Service Overview Dashboard (Grafana)

- Request rate, error rate, p50/p95/p99 latency per service
- Active DB connections per service
- Redis hit rate (cache vs. DB fallback in inventory)
- Kafka consumer lag per topic

### Business Dashboard

- Orders per hour (created vs. confirmed)
- Payment success rate
- Inventory reservation success rate
- Top 10 routes by error count

### Alerting Rules

| Alert | Condition | Severity |
|-------|-----------|----------|
| High error rate | `http_requests_total{status=~"5.."}` > 5% for 5m | critical |
| Circuit breaker open | `circuit_breaker_state == 1` for 1m | critical |
| High payment failure | payments_failed / payments_captured > 10% for 10m | warning |
| Kafka consumer lag | lag > 1000 for 5m | warning |
| Outbox accumulating | `outbox_pending_rows` > 500 for 5m | warning |
| Inventory out of stock | `inventory_insufficient_stock_total` rate > 10/min | info |

---

## 8.5 Structured Logging Improvements

Current: `log/slog` with JSON output, but inconsistent field names across services.

Standardize on a `pkg/logger` package:

```go
// pkg/logger/logger.go
func New(serviceName, appEnv string) *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: level(appEnv),
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey { a.Key = "ts" }
            if a.Key == slog.LevelKey { a.Key = "level" }
            if a.Key == slog.MessageKey { a.Key = "msg" }
            return a
        },
    })).With("service", serviceName)
}
```

Add `trace_id` and `span_id` to every log line via an `slog.Handler` wrapper that reads from the context:

```go
type traceHandler struct { slog.Handler }
func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
    span := trace.SpanFromContext(ctx)
    if span.IsRecording() {
        sc := span.SpanContext()
        r.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
        r.AddAttrs(slog.String("span_id", sc.SpanID().String()))
    }
    return h.Handler.Handle(ctx, r)
}
```

This makes logs and traces correlatable in any log aggregation tool (Loki, Datadog, etc.).

---

## 8.6 Health Checks

Each service should expose:
- `GET /healthz` → 200 if the process is alive (liveness)
- `GET /readyz` → 200 only if DB and Redis are reachable (readiness)

Update `docker-compose.yml` healthchecks to use these endpoints instead of TCP checks.

---

## Acceptance criteria

- A single buyer checkout is traceable as one distributed trace spanning: api-gateway → auth-service (token validate) → order-management-service → inventory-service → payment-service
- Jaeger UI shows the full trace with per-span latency
- Grafana shows real-time request rate and error rate per service
- A 5xx error rate spike fires an alert within 1 minute

---

## Estimated effort

2–3 weeks (tracing instrumentation: 1 week; metrics + Grafana dashboards: 1 week; logging standardization + alerting: 3–5 days). Can run in parallel with any other phase.
