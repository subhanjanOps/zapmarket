# ZapMarket — Master Gap Remediation Roadmap

> **Source:** `docs/architecture-gaps.md` (reviewed 2026-06-29)

This index links every identified gap to the plan that addresses it.
Execute plans in phase order. Each phase is independently shippable.

---

## Phase Order & Dependencies

```
Phase 1 — Critical Hotfixes          ← START HERE (blocking production safety)
    ↓
Phase 2 — Cart & Search              ← Unlocks real buyer flows
    ↓
Phase 3 — Seller Settlement          ← Required before real seller onboarding
    ↓
Phase 4 — Reviews, Promotions,       ← Feature completeness for launch
          Shipping, Notifications
    ↓
Phase 5 — Infrastructure Hardening   ← HA, secrets, load testing, contract tests
    ↓
Phase 6 — Architecture Migration     ← Long-running, parallel to above
```

---

## Gap → Plan Map

| Gap ID | Description | Severity | Plan |
|--------|-------------|----------|------|
| G1 | Synchronous checkout saga, no compensation | Critical | Phase 1 |
| G6 | Reservation TTL has no expiry worker | Critical | Phase 1 |
| G8 | API gateway: no circuit breaker or retry | High | Phase 1 |
| G3 | No Cart domain | Critical | Phase 2 |
| G2 | No search service | Critical | Phase 2 |
| G4 | No seller payout / settlement | High | Phase 3 |
| G5 | Single warehouse hardcoded | High | Phase 3 |
| G10 | No reviews, ratings, or returns domain | High | Phase 4 |
| G11 | No promotions / pricing engine | High | Phase 4 |
| G12 | No shipping / logistics integration | High | Phase 4 |
| G13 | Push and SMS notifications not wired | Medium | Phase 4 |
| G15 | No buyer wishlist | Medium | Phase 4 |
| G7 | No contract tests (gRPC + Kafka schemas) | High | Phase 5 |
| G16 | No admin analytics or reporting | Medium | Phase 5 |
| G17 | No load testing baseline | Medium | Phase 5 |
| G18 | No database high availability | Medium | Phase 5 |
| G19 | Secrets via env vars only | Medium | Phase 5 |
| G9 | Inconsistent architecture across services | High | Phase 6 |
| G14 | No seller bulk operations | Medium | Phase 6 |
| G20 | No mobile application | Medium | Separate project |

---

## Plan Files

| Phase | Plan File | Estimated Effort |
|-------|-----------|-----------------|
| 1 | `2026-06-29-phase1-critical-hotfixes.md` | 2.5 weeks |
| 2 | `2026-06-29-phase2-cart-and-search.md` | 4 weeks |
| 3 | `2026-06-29-phase3-seller-settlement.md` | 4 weeks |
| 4 | `2026-06-29-phase4-features.md` | 6 weeks |
| 5 | `2026-06-29-phase5-infrastructure.md` | 3 weeks |
| 6 | `2026-06-29-phase6-architecture-migration.md` | 8 weeks |
