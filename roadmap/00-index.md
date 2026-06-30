# ZapMarket Roadmap — Gap-Driven Implementation Plan

Source: `zapmarket-comparison-report.md` (2026-06-30)  
Benchmark: Amazon India

Plans are numbered sequentially by execution order. Each plan is self-contained: context, scope, tasks, and done criteria. Work them in order — later plans depend on earlier ones.

| # | File | Theme | Effort | Impact |
|---|---|---|---|---|
| 1 | [01-kafka-activation.md](01-kafka-activation.md) | Activate Kafka + Debezium (all sagas inert until done) | M | H |
| 2 | [02-saga-compensations.md](02-saga-compensations.md) | Inventory release on payment failure | S | H |
| 3 | [03-promotions-checkout.md](03-promotions-checkout.md) | Wire promotions-service into order creation | S | H |
| 4 | [04-delivery-agents.md](04-delivery-agents.md) | Delivery agent table, assignment, proof of delivery, reattempts | S→M | H |
| 5 | [05-cod.md](05-cod.md) | COD payment method end-to-end | M | H |
| 6 | [06-3pl-adapter.md](06-3pl-adapter.md) | Shiprocket HTTP adapter (CarrierClient impl) | M | H |
| 7 | [07-seller-compliance.md](07-seller-compliance.md) | TDS, GST on commission, bank/UPI payout, payout scheduler | M | H |
| 8 | [08-seller-kyc.md](08-seller-kyc.md) | Seller approval workflow + admin endpoints | S | M |
| 9 | [09-auth-hardening.md](09-auth-hardening.md) | Login rate limiting, account lockout, delivery address on orders | S | H |
| 10 | [10-logistics-ops.md](10-logistics-ops.md) | Reattempt scheduling, return pickup, reverse logistics | M | M |
| 11 | [11-reviews-returns.md](11-reviews-returns.md) | Return approval state machine, multi-item returns, rating aggregation | M | M |
| 12 | [12-razorpay.md](12-razorpay.md) | Razorpay gateway implementation | M | H |
| 13 | [13-reservation-sweeper.md](13-reservation-sweeper.md) | Expired reservation cleanup job | S | M |
| 14 | [14-ci-cd.md](14-ci-cd.md) | GitHub Actions CI + Prometheus alerting rules | S | M |
| 15 | [15-frontend-foundation.md](15-frontend-foundation.md) | Form validation, react-query, seller-ui component lib | M | M |
| 16 | [16-search-ranking.md](16-search-ranking.md) | Typesense boosting, trending backend, search ranking | S | M |
| 17 | [17-k8s.md](17-k8s.md) | Kubernetes + Helm manifests | L | H |
| 18 | [18-mfa.md](18-mfa.md) | TOTP/MFA at login for sellers and admins | M | M |
| 19 | [19-personalization.md](19-personalization.md) | Recently viewed, recommendations, user event tracking | L | M |
