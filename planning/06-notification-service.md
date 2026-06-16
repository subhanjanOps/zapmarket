# Stage 6 — Notification Service

Corresponds to checklist **Phase 11**.

## Goal

Build `notification-service` as a pure Kafka-consumer, no public REST surface, per
`design.md`. This is deliberately sequenced right before Kafka is actually turned on
(Stage 7) so the consumer code can be written and unit-tested against a fake/in-memory
event bus first, then wired to real Kafka in Stage 7 with minimal changes.

## Preconditions
- Stage 5 merged — Order Management exists and is the eventual source of
  `order.created` / `order.cancelled` events this service will consume.

## Tasks

### 6.1 Scaffolding
- [ ] Layout: `internal/domain`, `internal/service`, `internal/consumer` (replaces the
      usual `handler` dir — there's no inbound HTTP/gRPC handler layer here), `internal/notifiers`
      (email/SMS/push provider adapters behind an interface).
- [ ] `.env.example`: Kafka broker addr, consumer group ID, email/SMS provider creds
      (placeholders only), `HTTP_PORT` for a `/healthz` liveness endpoint only.

### 6.2 Consumer contract (build against an interface, not Kafka directly yet)
- [ ] Define `internal/domain/contracts.EventConsumer` (`Subscribe(topic string, handler
      func([]byte) error)`) so the actual transport (Kafka now, anything else later) is
      swappable.
- [ ] Implement a `FakeEventConsumer` for local tests that lets you publish a fake message
      and assert the right notifier fires.
- [ ] The real `pkg/kafka` consumer wrapper backing this interface is built/wired in
      Stage 7 — don't block this stage on Kafka infra being live.

### 6.3 Notifiers
- [ ] `internal/notifiers.EmailNotifier`, `SMSNotifier`, `PushNotifier` behind a common
      `Notifier` interface (`Send(ctx, recipient, payload) error`).
- [ ] Local/dev implementations log to stdout instead of calling a real provider SDK,
      matching the `FakePaymentGateway` pattern from Stage 4 — swap in real SDKs only
      when an account/credentials exist.

### 6.4 Event handling logic
- [ ] `order.created` → send order-confirmation email.
- [ ] `payment.processed` / `payment.failed` → send payment receipt / failure notice.
- [ ] `user.registered` → send welcome email.
- [ ] Dedup logic (per `design.md`'s Redis dedup-key note) — for this stage, dedup via an
      in-memory LRU keyed by event ID is acceptable; Redis-backed dedup is Stage 9.

## Out of scope
- Real Kafka wiring — Stage 7.
- Redis-backed rate limiting / dedup — Stage 9.
- Real provider SDK integration (SendGrid/Twilio/FCM etc.) — flag as a follow-up once
  the user has provider accounts; don't guess credentials or SDKs.

## Definition of done
- `FakeEventConsumer` test: publishing a fake `order.created` payload results in the
  stdout `EmailNotifier` logging a confirmation for the right recipient.
- Service starts and exposes `/healthz` even with zero real infra dependencies running.
