# Stage 12 — Production Readiness

Corresponds to checklist **Phase 18** and the final **Completion Criteria** section.

## Goal

Close the remaining security and reliability gaps, and do a final sign-off pass against
the checklist's completion criteria — the last stage before calling the platform
production-ready.

## Preconditions
- All prior stages (1-11) merged.

## Tasks

### 12.1 Security
- [ ] JWT rotation — support multiple active signing keys (`kid` header) so
      `JWT_SECRET_KEY` can be rotated without invalidating all outstanding tokens
      instantly.
- [ ] Refresh token revocation — this is now straightforward given Stage 9's
      `auth:blacklist:{jti}` Redis key; wire an explicit "log out everywhere" /
      admin-revoke endpoint that populates it.
- [ ] Secrets management — move from `.env` files (fine for local dev) to the cluster's
      Secret mechanism in all non-local environments (already scaffolded in Stage 11;
      this task is about making sure no service still reads a plaintext `.env` in
      anything but local dev mode).
- [ ] TLS everywhere — internal gRPC traffic between services should use TLS (mTLS if
      feasible) once outside a trusted local-dev network; terminate public TLS at the
      Gateway (or Ingress, depending on Stage 11's final topology).

### 12.2 Reliability
- [ ] Retry policies — standardize retry-with-backoff for all inter-service gRPC calls
      (the Stage 7 Kafka consumer retry logic was a first instance of this; generalize it
      into a shared `pkg/grpcx` client interceptor).
- [ ] Circuit breakers — extend the Stage 8 gateway-only circuit breaker to service-to-
      service calls (Order → Inventory, Order → Payment) so a degraded downstream doesn't
      cascade through the saga.
- [ ] Dead letter queues — replace Stage 7's "log and skip after N retries" Kafka
      consumer behavior with a real DLQ topic per consumer, plus a basic reprocessing
      tool/runbook.
- [ ] Graceful shutdown — audit every service's `main.go` for proper `context` cancellation
      and connection draining on SIGTERM (important for k8s rolling deploys from Stage 11
      to not drop in-flight requests).

### 12.3 Final completion criteria sign-off

Walk the checklist's existing "Completion Criteria" list and verify each item against
the actual repo state, not just against intent:
- [ ] Shared infrastructure packages complete
- [ ] Centralized protobuf ownership
- [ ] Outbox pattern implemented
- [ ] Saga orchestration implemented
- [ ] OpenTelemetry integrated
- [ ] Kubernetes deployment ready
- [ ] CI/CD operational
- [ ] Redis caching enabled
- [ ] API Gateway operational
- [ ] Health/readiness endpoints implemented
- [ ] End-to-end order flow passes integration tests

## Out of scope
- New feature work — this stage is exclusively hardening what already exists across
  Stages 1-11.

## Definition of done
- Every box above is checked and independently verifiable (link a test, a runbook, or a
  dashboard for each, not just a code change).
- A documented incident-response runbook exists for at least: payment gateway outage,
  Kafka consumer lag spike, and inventory/payment ledger drift.
