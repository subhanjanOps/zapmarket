# Microservice Template

## Standard Service Structure

```text
service-name/

cmd/
└── server/

internal/

├── domain/
│   ├── entities/
│   ├── valueobjects/
│   ├── repositories/
│   └── services/
│
├── application/
│   ├── usecases/
│   ├── dto/
│   └── ports/
│
├── infrastructure/
│   ├── postgres/
│   ├── redis/
│   ├── kafka/
│   ├── grpc/
│   └── external/
│
├── interfaces/
│   ├── http/
│   ├── grpc/
│   └── consumers/
│
└── shared/

configs/
migrations/
tests/
```

---

# Service Responsibilities

Each service should own:

* Domain Logic
* Database
* Events
* APIs

---

# Handler Flow

```text
HTTP Request
    ↓
Handler
    ↓
Use Case
    ↓
Repository
    ↓
Database
```

---

# Kafka Consumer Flow

```text
Kafka Consumer
    ↓
Event Validation
    ↓
Use Case
    ↓
Repository
```

---

# gRPC Flow

```text
gRPC Service
    ↓
DTO Mapping
    ↓
Use Case
    ↓
Domain
```

---

# Repository Contract Example

```go
type OrderRepository interface {
	GetByID(
		ctx context.Context,
		id string,
	) (*Order, error)

	Save(
		ctx context.Context,
		order *Order,
	) error
}
```

---

# Use Case Example

```go
type CreateOrderUseCase struct {
	orderRepo OrderRepository
	publisher EventPublisher
}
```

Use cases orchestrate business workflows.

---

# Testing Structure

```text
tests/

├── unit/
├── integration/
├── contract/
└── e2e/
```

Coverage targets:

| Area            | Target |
| --------------- | ------ |
| Domain Logic    | 90%+   |
| Use Cases       | 85%+   |
| Service Overall | 70%+   |

---

# Required Non-Functional Requirements

Every service must support:

* Health Checks
* Readiness Checks
* Metrics
* Tracing
* Structured Logging
* Graceful Shutdown
* Configuration Validation

These are mandatory for production readiness.
