# Architecture Principles

## Architectural Style

ZapMarket follows:

* Clean Architecture
* Domain Driven Design
* Event Driven Architecture
* Microservices Architecture

---

# Dependency Rule

Dependencies always flow inward.

Outer layers depend on inner layers.

Inner layers never depend on:

* HTTP
* Kafka
* PostgreSQL
* Redis
* gRPC
* Frameworks

---

# Layer Structure

```text
internal/

├── domain/
├── application/
├── infrastructure/
├── interfaces/
└── shared/
```

---

# Domain Layer

Contains:

* Entities
* Value Objects
* Domain Services
* Repository Interfaces

Domain must be independent.

Forbidden:

* SQL
* Kafka
* HTTP
* Redis
* Framework code

---

# Application Layer

Contains:

* Use Cases
* DTOs
* Ports
* Orchestration

Business workflows live here.

---

# Infrastructure Layer

Contains:

* PostgreSQL
* Redis
* Kafka
* AWS
* External APIs

Infrastructure implements contracts defined by inner layers.

---

# Interface Layer

Contains:

* REST Handlers
* gRPC Services
* Kafka Consumers

Responsibilities:

1. Parse request
2. Validate request
3. Call use case
4. Return response

Nothing else.

---

# Event Driven Principles

Events represent business facts.

Examples:

```text
order.created.v1
payment.completed.v1
inventory.reserved.v1
```

Events must be:

* Immutable
* Versioned
* Backward Compatible

---

# Service Boundaries

Each service owns:

* Its database
* Its domain
* Its business rules

Never allow direct database access between services.

Communication must happen through:

* gRPC
* Kafka
* APIs

---

# Dependency Injection

Always use constructor injection.

Bad:

```go
global.DB
singleton.GetDB()
```

Good:

```go
type OrderService struct {
	repo OrderRepository
}
```

Dependencies must always be explicit.
