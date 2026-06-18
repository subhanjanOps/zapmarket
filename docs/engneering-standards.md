# Engineering Standards

## Guiding Principle

Code is read more often than it is written.

Prioritize:

* Readability
* Maintainability
* Testability
* Scalability
* Reliability

Avoid clever solutions that increase cognitive load.

---

# Development Philosophy

Before implementing any feature, evaluate:

1. Is this the simplest possible solution?
2. Is the design scalable?
3. Is the implementation testable?
4. Will another engineer understand this six months later?
5. Does it introduce technical debt?

If the answer is unclear, redesign before implementation.

---

# SOLID Principles

## Single Responsibility Principle

Every:

* Package
* File
* Struct
* Interface
* Function

Should have one reason to change.

---

## Open Closed Principle

Systems should be open for extension and closed for modification.

Prefer:

* Interfaces
* Composition
* Strategy Pattern

Avoid repeated modification of stable code.

---

## Liskov Substitution Principle

Implementations must fully satisfy interface contracts.

Never surprise callers.

---

## Interface Segregation Principle

Prefer smaller interfaces.

Bad:

```go
type UserRepository interface {
	Create()
	Update()
	Delete()
	Get()
	SendEmail()
}
```

Good:

```go
type UserReader interface {
	GetByID(ctx context.Context, id string)
}

type UserWriter interface {
	Create(ctx context.Context, user User)
}
```

---

## Dependency Inversion Principle

Depend on abstractions.

Never depend directly on infrastructure implementations.

---

# Error Handling

Never ignore errors.

Bad:

```go
result, _ := repo.Get()
```

Good:

```go
result, err := repo.Get()
if err != nil {
	return err
}
```

Wrap errors:

```go
return fmt.Errorf("fetch user: %w", err)
```

---

# Logging Standards

Use structured logs only.

Good:

```go
logger.Info(
	"order created",
	"order_id", orderID,
)
```

Bad:

```go
logger.Infof("order %s created", orderID)
```

---

# Security Standards

Never:

* Hardcode credentials
* Commit secrets
* Log passwords
* Log tokens

Always validate:

* Input
* Authentication
* Authorization

---

# Observability

All services must support:

* Structured Logging
* Metrics
* Distributed Tracing

Required context:

* trace_id
* request_id
* correlation_id

---

# Refactoring Rules

Whenever touching code:

1. Remove duplication.
2. Simplify logic.
3. Improve naming.
4. Improve testability.
5. Reduce coupling.

Always leave the codebase cleaner than before.
