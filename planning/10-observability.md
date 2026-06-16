# Stage 10 — Observability

Corresponds to checklist **Phase 15**.

## Goal

Add metrics, tracing, and dashboards across all six services plus the gateway, so the
saga flow built in Stages 5-7 can actually be debugged in production instead of only
via logs.

## Preconditions
- Stages 1-9 merged — observability is applied across a stable set of services rather
  than retrofitted piecemeal into each one as it's built.

## Tasks

### 10.1 Prometheus metrics
- [ ] Add `pkg/metrics` with standard HTTP/gRPC middleware exposing request count,
      latency histograms, and error rate per route/RPC — wire into every service's
      `pkg/httpx` and `pkg/grpcx` bootstrap so it's automatic, not opt-in per service.
- [ ] Business metrics specific to the saga: orders created/confirmed/cancelled counts,
      inventory reservation success/failure rate, payment success/failure rate.
- [ ] Add a `prometheus` service to `docker-compose.yml` scraping each service's
      `/metrics` endpoint.

### 10.2 OpenTelemetry tracing
- [ ] Instrument the gRPC client/server interceptors in `pkg/grpcx` with OTel spans, and
      HTTP middleware in `pkg/httpx`, so a single checkout request's trace spans Gateway →
      Order → Inventory → Payment in one view.
- [ ] Correlation ID from Stage 8's gateway middleware should map to/become the OTel
      trace ID, not a separate parallel concept.
- [ ] Add a Jaeger or Tempo collector to `docker-compose.yml`.

### 10.3 Grafana dashboards
- [ ] Add `grafana` service wired to the Prometheus data source.
- [ ] Build one dashboard per service plus one "checkout saga health" dashboard
      combining the business metrics from 10.1.

### 10.4 Kafka metrics
- [ ] Export consumer lag and throughput per topic/consumer-group (the Kafka client
      library chosen in Stage 7 likely has Prometheus exporters available — use those
      rather than hand-rolling).

## Out of scope
- Log aggregation (ELK/Loki) — not in the checklist's Phase 15 scope; flag as a possible
  future addition if the user wants centralized log search later.

## Definition of done
- A trace for a single checkout request is visible end-to-end in Jaeger/Tempo, showing
  the Order → Inventory → Payment call chain with accurate timing.
- Grafana's checkout-saga dashboard shows live order/payment/inventory counters during a
  manual test run.
