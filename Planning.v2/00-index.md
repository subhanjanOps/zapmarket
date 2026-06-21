# ZapMarket — Phase Planning v2
**Created:** 2026-06-21  
**Baseline:** All six backend services + three UIs at production-review quality (see `reviews/2026-06-21-full-review.md`)

## Phases

| # | Name | Goal | Est. Complexity |
|---|------|------|-----------------|
| 1 | [Testing Foundation](01-testing-foundation.md) | Baseline unit + integration coverage before further change | High |
| 2 | [Outbox / Event Pipeline](02-outbox-event-pipeline.md) | Wire Debezium CDC so outbox rows actually reach Kafka | Medium |
| 3 | [Auth Hardening](03-auth-hardening.md) | Refresh-token BFF flow, email verification, password reset, rate limiting | High |
| 4 | [Import Service](04-import-service.md) | Async CSV import with MinIO + job table + batch worker | Medium |
| 5 | [Search & Discovery](05-search-discovery.md) | Elasticsearch product search, category facets, real-time sync | High |
| 6 | [Payment Gateway](06-payment-gateway.md) | Replace FakeGateway with Razorpay/Stripe, webhook lifecycle | High |
| 7 | [Seller & Buyer UX](07-seller-buyer-ux.md) | Order tracking, shipment, reviews, notifications UI | High |
| 8 | [Observability](08-observability.md) | OpenTelemetry traces, structured metrics, dashboards | Medium |

## Sequencing rationale

```
Phase 1 (Tests) ──► Phase 2 (Events) ──► Phase 3 (Auth)
                         │
                         ▼
                    Phase 4 (Import) ──► Phase 5 (Search)
                    Phase 6 (Payment)
                    Phase 7 (UX) ──── depends on 2, 3, 6
                    Phase 8 (Obs) ──── can start any time
```

- **Phase 1 first:** tests give a safety net for all subsequent changes.
- **Phase 2 before Phase 7:** notifications, order tracking, and payment confirmations all depend on events actually flowing through Kafka.
- **Phase 6 before buyer checkout UX:** buyers can't complete purchases until a real payment gateway is wired.
