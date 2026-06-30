# Plan 14 — CI/CD and Alerting

**Effort:** S | **Impact:** M | **Depends on:** none

## Context
Zero automated pipelines exist. No `.github/workflows/` directory. Prometheus has one dashboard but zero alerting rules. Every PR merges untested.

## Scope
- GitHub Actions: per-service `go test ./...` on PR
- GitHub Actions: docker build verification on PR
- Prometheus: basic alerting rules for the most critical conditions

## Tasks

### GitHub Actions CI
- [ ] Create `.github/workflows/ci.yml`
- [ ] Matrix strategy: one job per Go service (`auth-service`, `product-catalog-service`, `order-management-service`, `payment-service`, `inventory-service`, `cart-service`, `currency-service`, `settlement-service`, `logistics-service`, `promotions-service`, `review-return-service`, `wishlist-service`, `notification-service`, `import-service`, `api-gateway`)
- [ ] Each job: `go test ./... -race -count=1` from the service directory
- [ ] Separate job: `go work sync && go build ./...` at workspace root to catch module issues
- [ ] Next.js jobs: `npm ci && npm run build` for each UI (buyer-ui, seller-ui, admin-ui, backoffice-ui)
- [ ] Cache `~/.cache/go` and `node_modules` across runs

### Prometheus alerting rules
- [ ] Create `monitoring/prometheus/alerts.yml`
- [ ] Alert: `ServiceDown` — any scrape target up == 0 for > 2 minutes
- [ ] Alert: `HighErrorRate` — HTTP 5xx rate > 1% over 5 minutes per service
- [ ] Alert: `OutboxLag` — `outbox_unpublished_count > 100` for > 5 minutes (requires services to expose this metric)
- [ ] Alert: `InventoryRedisDown` — inventory-service health check fails (proxy via blackbox exporter or custom metric)
- [ ] Alert: `KafkaConsumerLag` — consumer group lag > 1000 messages
- [ ] Add `rule_files: [alerts.yml]` to `monitoring/prometheus.yml`
- [ ] Configure Grafana alertmanager datasource (or use Grafana alerting directly from existing dashboard)

## Done criteria
- PR to any service triggers `go test ./...` automatically; failure blocks merge
- All 5 Prometheus alerts visible in Grafana alert rules panel
- `ServiceDown` alert fires within 2 minutes of a service stopping in `docker compose`
